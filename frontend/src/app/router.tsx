import { createBrowserRouter, Navigate } from "react-router-dom";
import LoginPage from "../pages/LoginPage";
import RegisterPage from "../pages/RegisterPage";
import WatchlistPage from "../pages/WatchlistPage";
import SettingsTinkoffPage from "../pages/SettingsTinkoffPage";
import BrokerPage from "../pages/BrokerPage";
import InstrumentPage from "../pages/InstrumentPage";
import AlertsPage from "../pages/AlertsPage";
import AdminMLPage from "../pages/AdminMLPage";
import { ProtectedRoute } from "./ProtectedRoute";
import { AppShell } from "./AppShell";

export const router = createBrowserRouter([
  {
    path: "/login",
    element: <LoginPage />,
  },
  {
    path: "/register",
    element: <RegisterPage />,
  },
  {
    path: "/",
    element: <Navigate to="/watchlist" replace />,
  },
  {
    element: <ProtectedRoute />,
    children: [
      {
        element: <AppShell />,
        children: [
          {
            path: "watchlist",
            element: <WatchlistPage />,
          },
          {
            path: "broker",
            element: <BrokerPage />,
          },
          {
            path: "settings/tinkoff",
            element: <SettingsTinkoffPage />,
          },
          {
            path: "instruments/:id",
            element: <InstrumentPage />,
          },
          {
            path: "alerts",
            element: <AlertsPage />,
          },
          {
            path: "admin",
            element: <ProtectedRoute permission="ml_admin" />,
            children: [
              {
                index: true,
                element: <AdminMLPage />,
              },
            ],
          },
        ],
      },
    ],
  },
]);
