package httpserver

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"invest/backend/internal/http/handlers"
	"invest/backend/internal/service"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap ResponseWriter to capture status code
		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		log.Printf("%s %s %d %s", r.Method, r.URL.Path, wrapped.status, time.Since(start))
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %v", err)
				handlers.WriteError(w, http.StatusInternalServerError, "internal_error", "An unexpected error occurred", nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func authMiddleware(auth *service.AuthService, appEnv string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if auth == nil {
				if appEnv == "test" {
					ctx := context.WithValue(r.Context(), "user_id", int64(1))
					ctx = context.WithValue(ctx, "user_email", "test@example.local")
					ctx = context.WithValue(ctx, "user_role", "super_admin")
					ctx = context.WithValue(ctx, "user_permissions", []string{"market_access", "ml_admin", "user_admin"})
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				handlers.WriteError(w, http.StatusServiceUnavailable, "auth_unavailable", "auth service is not configured", nil)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			token := parts[1]
			claims, err := auth.ValidateTokenClaims(token)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
			ctx = context.WithValue(ctx, "user_email", claims.Email)
			ctx = context.WithValue(ctx, "user_role", claims.Role)
			ctx = context.WithValue(ctx, "user_permissions", claims.Permissions)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func requirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value("user_role").(string)
			if role == "super_admin" || role == "admin" {
				next.ServeHTTP(w, r)
				return
			}
			permissions, _ := r.Context().Value("user_permissions").([]string)
			for _, item := range permissions {
				if item == permission {
					next.ServeHTTP(w, r)
					return
				}
			}
			handlers.WriteError(w, http.StatusForbidden, "forbidden", "permission required: "+permission, nil)
		})
	}
}
