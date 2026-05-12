import React, { useEffect, useState } from "react";
import { Navigate, Outlet } from "react-router-dom";
import { api } from "../shared/api/client";

interface ProtectedRouteProps {
  permission?: string;
}

interface MeResponse {
  role?: string;
  permissions?: string[];
}

export const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ permission }) => {
  const token = localStorage.getItem("access_token");
  const [me, setMe] = useState<MeResponse | null>(null);
  const [loading, setLoading] = useState(Boolean(permission && token));

  useEffect(() => {
    if (!permission || !token) return;
    api.get<MeResponse>("/api/v1/me")
      .then(setMe)
      .catch(() => setMe(null))
      .finally(() => setLoading(false));
  }, [permission, token]);

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  if (loading) {
    return <div className="page"><div className="loading-panel">Проверяем доступ...</div></div>;
  }

  if (permission) {
    const allowed = me?.role === "super_admin" || me?.role === "admin" || me?.permissions?.includes(permission);
    if (!allowed) {
      return <Navigate to="/watchlist" replace />;
    }
  }

  return <Outlet />;
};
