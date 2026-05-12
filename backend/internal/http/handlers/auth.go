package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"invest/backend/internal/app"
	"invest/backend/internal/service"
)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func Register(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req registerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body", nil)
			return
		}

		if req.Email == "" || req.Password == "" {
			WriteError(w, http.StatusBadRequest, "validation_error", "email and password are required", nil)
			return
		}

		user, err := container.Services.Auth.Register(r.Context(), req.Email, req.Password)
		if err != nil {
			if errors.Is(err, service.ErrUserAlreadyExists) {
				WriteError(w, http.StatusConflict, "user_already_exists", err.Error(), nil)
				return
			}
			WriteError(w, http.StatusInternalServerError, "registration_failed", err.Error(), nil)
			return
		}

		WriteJSON(w, http.StatusCreated, user)
	}
}

func Login(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body", nil)
			return
		}

		accessToken, refreshToken, err := container.Services.Auth.Login(r.Context(), req.Email, req.Password)
		if err != nil {
			if errors.Is(err, service.ErrInvalidCredentials) {
				WriteError(w, http.StatusUnauthorized, "invalid_credentials", err.Error(), nil)
				return
			}
			WriteError(w, http.StatusInternalServerError, "login_failed", err.Error(), nil)
			return
		}

		WriteJSON(w, http.StatusOK, loginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		})
	}
}

func Refresh(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req refreshRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body", nil)
			return
		}

		accessToken, refreshToken, err := container.Services.Auth.Refresh(r.Context(), req.RefreshToken)
		if err != nil {
			WriteError(w, http.StatusUnauthorized, "invalid_token", err.Error(), nil)
			return
		}

		WriteJSON(w, http.StatusOK, loginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		})
	}
}

func Logout(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req logoutRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body", nil)
			return
		}

		err := container.Services.Auth.Logout(r.Context(), req.RefreshToken)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "logout_failed", err.Error(), nil)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}


func Me(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// User ID will be injected by middleware
		userID, ok := r.Context().Value("user_id").(int64)
		if !ok {
			WriteError(w, http.StatusUnauthorized, "unauthorized", "user not found in context", nil)
			return
		}

		user, err := container.Services.Auth.GetUserByID(r.Context(), userID)
		if err != nil {
			WriteError(w, http.StatusUnauthorized, "unauthorized", "user not found", nil)
			return
		}

		WriteJSON(w, http.StatusOK, user)
	}
}
