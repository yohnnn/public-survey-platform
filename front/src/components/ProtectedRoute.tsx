import { Link } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import type { ReactNode } from "react";

export function ProtectedRoute({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useAuth();
  if (isAuthenticated) return children;

  return (
    <section className="page-head">
      <div>
        <h1>Нужен вход</h1>
        <p>Авторизуйтесь, чтобы пользоваться этим разделом.</p>
      </div>
      <Link className="button" to="/auth">
        Войти
      </Link>
    </section>
  );
}
