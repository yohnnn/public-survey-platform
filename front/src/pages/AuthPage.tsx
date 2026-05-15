import { FormEvent, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { useToast } from "../components/Toast";

export function AuthPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const isRegister = searchParams.get("mode") === "register";
  const [busy, setBusy] = useState(false);
  const { api, setTokens, loadMe } = useAuth();
  const toast = useToast();
  const navigate = useNavigate();
  const title = useMemo(() => (isRegister ? "Создайте аккаунт" : "Вернитесь к опросам"), [isRegister]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const payload: Record<string, string | number> = Object.fromEntries([...form.entries()].map(([key, value]) => [key, String(value).trim()]));
    if (isRegister) payload.birthYear = Number(payload.birthYear);

    setBusy(true);
    try {
      const response = isRegister ? await api.register(payload) : await api.login(payload);
      setTokens(response.tokens);
      await loadMe(true);
      toast(isRegister ? "Аккаунт создан." : "Вы вошли.");
      navigate("/feed");
    } catch (error) {
      toast(error instanceof Error ? error.message : "Не удалось выполнить вход.", "error");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="auth-layout page-view">
      <div className="hero-copy compact-copy">
        <span className="eyebrow">Аккаунт</span>
        <h1>{title}</h1>
        <p>{isRegister ? "Аккаунт нужен для создания опросов, голосования и подписок." : "Войдите, чтобы голосовать, создавать опросы и видеть свою ленту подписок."}</p>
      </div>
      <section className="card stack">
        <div className="tabbar">
          <button className={!isRegister ? "active" : ""} type="button" onClick={() => setSearchParams({})}>
            Вход
          </button>
          <button className={isRegister ? "active" : ""} type="button" onClick={() => setSearchParams({ mode: "register" })}>
            Регистрация
          </button>
        </div>
        <form className="stack" onSubmit={handleSubmit}>
          <label>
            Email
            <input name="email" type="email" autoComplete="email" required />
          </label>
          <label>
            Пароль
            <input name="password" type="password" autoComplete={isRegister ? "new-password" : "current-password"} minLength={isRegister ? 6 : undefined} required />
          </label>
          {isRegister ? (
            <>
              <label>
                Никнейм
                <input name="nickname" autoComplete="nickname" required />
              </label>
              <div className="grid-two">
                <label>
                  Страна
                  <input name="country" placeholder="RU" required />
                </label>
                <label>
                  Год рождения
                  <input name="birthYear" type="number" min="1900" max="2100" required />
                </label>
              </div>
              <label>
                Пол
                <select name="gender" required defaultValue="male">
                  <option value="male">Мужской</option>
                  <option value="female">Женский</option>
                  <option value="other">Другой</option>
                </select>
              </label>
            </>
          ) : null}
          <button type="submit" disabled={busy}>
            {busy ? "Отправка..." : isRegister ? "Создать аккаунт" : "Войти"}
          </button>
        </form>
      </section>
    </section>
  );
}
