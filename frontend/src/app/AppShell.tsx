import { BarChart3, Bell, KeyRound, LineChart, LogOut, Settings2, WalletCards } from "lucide-react";
import { Link, useLocation } from "react-router-dom";
import { useEffect, useMemo, useState } from "react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { api } from "../shared/api/client";
import { ToastProvider } from "../shared/ui/Toast";
import { NotificationManager } from "./NotificationManager";

const navItems = [
  { to: "/watchlist", label: "Котировки", icon: LineChart },
  { to: "/broker", label: "Счета", icon: WalletCards },
  { to: "/alerts", label: "Уведомления", icon: Bell },
  { to: "/settings/tinkoff", label: "T-Invest API", icon: KeyRound },
];

const adminNavItem = { to: "/admin", label: "Admin ML", icon: BarChart3 };

interface MeResponse {
  email: string;
  role?: string;
  permissions?: string[];
}

function AppShellContent() {
  const navigate = useNavigate();
  const location = useLocation();
  const [me, setMe] = useState<MeResponse | null>(null);
  const [profileOpen, setProfileOpen] = useState(false);

  const isInstrumentPage = location.pathname.startsWith("/instruments/");

  useEffect(() => {
    api.get<MeResponse>("/api/v1/me").then(setMe).catch(() => setMe(null));
  }, []);

  const visibleNavItems = useMemo(() => {
    const permissions = new Set(me?.permissions || []);
    const isAdmin = me?.role === "super_admin" || permissions.has("ml_admin") || permissions.has("user_admin");
    return isAdmin ? [...navItems, adminNavItem] : navItems;
  }, [me]);

  const handleLogout = () => {
    localStorage.removeItem("access_token");
    localStorage.removeItem("refresh_token");
    navigate("/login", { replace: true });
  };

  const userInitial = me?.email ? me.email.charAt(0).toUpperCase() : "U";

  return (
    <div className={`app-shell ${isInstrumentPage ? "chart-shell" : "workbench-shell"}`}>
      <header className={`app-topbar ${isInstrumentPage ? "chart-topbar" : "app-sidebar"}`} style={isInstrumentPage ? { borderBottom: "none" } : {}}>
        {!isInstrumentPage && (
          <div className="brand">
            <div className="brand-mark">
              <Settings2 size={18} aria-hidden="true" />
            </div>
            <div>
              <strong>Invest Terminal</strong>
              <span>Графики и прогнозы</span>
            </div>
          </div>
        )}

        {isInstrumentPage ? (
          <div id="chart-topbar-portal" style={{ display: 'flex', alignItems: 'center', flex: 1, minWidth: 0, gap: '4px' }} />
        ) : (
          <nav className="topbar-nav" aria-label="Основная навигация">
            {visibleNavItems.map((item) => {
              const Icon = item.icon;
              return (
                <NavLink key={item.to} to={item.to} className={({ isActive }) => (isActive ? "nav-item active" : "nav-item")}>
                  <Icon size={18} aria-hidden="true" />
                  <span>{item.label}</span>
                </NavLink>
              );
            })}
          </nav>
        )}

        <div className="profile-menu">
          <button type="button" className="profile-menu-button styled-profile-button" onClick={() => setProfileOpen((value) => !value)} aria-expanded={profileOpen} aria-label="Открыть профиль">
            <span className="profile-avatar">{userInitial}</span>
            {!isInstrumentPage && (
              <span className="profile-label">
                <strong>{me?.email || "Пользователь"}</strong>
                <small>{me?.role || "user"}</small>
              </span>
            )}
          </button>
          {profileOpen && (
            <div className="profile-dropdown">
              <div className="profile-dropdown-user">
                <strong>{me?.email || "Пользователь"}</strong>
                <span>{me?.role || "user"}</span>
              </div>
              
              {isInstrumentPage && (
                <nav className="profile-nav profile-nav-main">
                  <div className="profile-nav-header">Навигация</div>
                  {visibleNavItems.map((item) => {
                    const Icon = item.icon;
                    return (
                      <Link key={item.to} to={item.to} className="profile-link" onClick={() => setProfileOpen(false)}>
                        <Icon size={17} aria-hidden="true" />
                        <span>{item.label}</span>
                      </Link>
                    );
                  })}
                </nav>
              )}

              <nav className="profile-nav">
                <div className="profile-nav-header">Аккаунт</div>
                <button type="button" onClick={handleLogout} className="profile-link logout-button">
                  <LogOut size={17} aria-hidden="true" />
                  <span>Выход</span>
                </button>
              </nav>
            </div>
          )}
        </div>
      </header>

      <main className="main-content" style={isInstrumentPage ? { padding: 0 } : {}}>
        <Outlet />
      </main>
    </div>
  );
}

export function AppShell() {
  return (
    <ToastProvider>
      <NotificationManager />
      <AppShellContent />
    </ToastProvider>
  );
}
