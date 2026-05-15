import { Menu, Plus, UserRound } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { useToast } from "./Toast";

const navItems = [
  { href: "/", label: "Главная" },
  { href: "/feed", label: "Лента" },
  { href: "/trending", label: "Тренды" },
  { href: "/tags", label: "Теги" },
  { href: "/following", label: "Подписки", auth: true },
  { href: "/create", label: "Создать", auth: true },
  { href: "/me", label: "Профиль", auth: true },
];

export function Layout() {
  const [menuOpen, setMenuOpen] = useState(false);
  const { me, isAuthenticated, logout, loadMe } = useAuth();
  const toast = useToast();
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    setMenuOpen(false);
    void loadMe(false);
  }, [loadMe, location.pathname]);

  async function handleLogout() {
    await logout();
    toast("Вы вышли.");
    navigate("/");
  }

  return (
    <div className="shell">
      <header className={`site-header ${menuOpen ? "menu-open" : ""}`}>
        <Link className="brand" to="/">
          <span className="brand-mark">PS</span>
          <span>
            <strong>Public Survey</strong>
            <small>опросы для всех</small>
          </span>
        </Link>

        <button className="icon-button mobile-menu-button" type="button" aria-label="Открыть меню" onClick={() => setMenuOpen((value) => !value)}>
          <Menu size={20} />
        </button>

        <nav className="main-nav">
          {navItems
            .filter((item) => !item.auth || isAuthenticated)
            .map((item) => (
              <NavLink key={item.href} to={item.href} end={item.href === "/"}>
                {item.label}
              </NavLink>
            ))}
        </nav>

        <div className="session-actions">
          {isAuthenticated ? (
            <>
              <Link className="button secondary compact" to="/create">
                <Plus size={17} />
                Создать
              </Link>
              <Link className="button secondary compact" to="/me">
                <UserRound size={17} />
                {me?.nickname || "Профиль"}
              </Link>
              <button className="ghost compact" type="button" onClick={handleLogout}>
                Выйти
              </button>
            </>
          ) : (
            <>
              <Link className="button secondary compact" to="/auth">
                Войти
              </Link>
              <Link className="button compact" to="/auth?mode=register">
                Регистрация
              </Link>
            </>
          )}
        </div>
      </header>

      <main className="app" tabIndex={-1}>
        <Outlet />
      </main>
    </div>
  );
}
