import { FormEvent, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { LoadingState } from "../components/LoadingState";
import { PollList } from "../components/PollCard";
import { useToast } from "../components/Toast";
import type { FeedItem } from "../types/domain";
import { initials } from "../utils/format";

export function MePage() {
  const { api, me, setMe, loadMe } = useAuth();
  const toast = useToast();
  const [polls, setPolls] = useState<FeedItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;
    Promise.all([loadMe(true), api.feed("/v1/feed/me", "limit=20", true).catch(() => ({ items: [] }))])
      .then(([, pollsResponse]) => active && setPolls(pollsResponse.items || []))
      .finally(() => active && setLoading(false));
    return () => {
      active = false;
    };
  }, [api, loadMe]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    const birthYear = String(data.get("birthYear") || "").trim();
    try {
      const response = await api.updateMe({
        email: String(data.get("email") || "").trim(),
        nickname: String(data.get("nickname") || "").trim(),
        country: String(data.get("country") || "").trim(),
        gender: String(data.get("gender") || "").trim(),
        ...(birthYear ? { birthYear: Number(birthYear) } : {}),
      });
      setMe(response.user);
      toast("Профиль обновлён.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Не удалось обновить профиль.", "error");
    }
  }

  if (loading || !me) return <LoadingState title="Мой профиль" />;

  return (
    <div className="page-view stack">
      <section className="page-head">
        <div>
          <h1>Мой профиль</h1>
          <p>Личные данные и ваши опубликованные опросы.</p>
        </div>
        <Link className="button" to="/create">Создать опрос</Link>
      </section>
      <div className="split">
        <section className="card stack">
          <div className="profile-avatar">{initials(me.nickname || me.email)}</div>
          <h2>{me.nickname || "-"}</h2>
          <form className="stack" onSubmit={submit}>
            <label>Email<input name="email" type="email" defaultValue={me.email || ""} /></label>
            <label>Никнейм<input name="nickname" defaultValue={me.nickname || ""} /></label>
            <div className="grid-two">
              <label>Страна<input name="country" defaultValue={me.country || ""} /></label>
              <label>Год рождения<input name="birthYear" type="number" defaultValue={me.birthYear || ""} /></label>
            </div>
            <label>Пол<input name="gender" defaultValue={me.gender || ""} /></label>
            <button type="submit">Сохранить профиль</button>
          </form>
        </section>
        <section className="stack">
          <div className="card">
            <h2>Мои опросы</h2>
            <p className="muted">Здесь отображается лента ваших публикаций.</p>
          </div>
          <PollList items={polls} />
        </section>
      </div>
    </div>
  );
}
