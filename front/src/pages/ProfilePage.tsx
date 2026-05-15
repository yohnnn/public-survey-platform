import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { ErrorState } from "../components/ErrorState";
import { LoadingState } from "../components/LoadingState";
import { PollList } from "../components/PollCard";
import { useToast } from "../components/Toast";
import type { FeedItem, PublicUserProfile } from "../types/domain";
import { initials, toCount } from "../utils/format";

export function ProfilePage() {
  const { id = "" } = useParams();
  const { api, isAuthenticated, me } = useAuth();
  const toast = useToast();
  const [profile, setProfile] = useState<PublicUserProfile | null>(null);
  const [polls, setPolls] = useState<FeedItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    setLoading(true);
    Promise.all([api.profile(id, isAuthenticated), api.feed(`/v1/users/${encodeURIComponent(id)}/polls`, "limit=20")])
      .then(([profileResponse, pollsResponse]) => {
        if (!active) return;
        setProfile(profileResponse.profile);
        setPolls(pollsResponse.items || []);
        setError("");
      })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Не удалось загрузить профиль."))
      .finally(() => active && setLoading(false));
    return () => {
      active = false;
    };
  }, [api, id, isAuthenticated]);

  async function follow() {
    if (!profile) return;
    try {
      await api.follow(profile.id, profile.isFollowing);
      setProfile({ ...profile, isFollowing: !profile.isFollowing });
      toast(profile.isFollowing ? "Вы отписались." : "Вы подписались.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Не удалось изменить подписку.", "error");
    }
  }

  if (loading) return <LoadingState title="Профиль" />;
  if (error || !profile) return <ErrorState message={error} />;

  const isOwn = me?.id === profile.id;

  return (
    <div className="page-view stack">
      <section className="page-head">
        <div>
          <h1>{profile.nickname || "Пользователь"}</h1>
          <p>Публичный профиль и опубликованные опросы.</p>
        </div>
        {!isAuthenticated ? <Link className="button" to="/auth">Войти, чтобы подписаться</Link> : isOwn ? <span className="chip">Это вы</span> : <button type="button" onClick={follow}>{profile.isFollowing ? "Отписаться" : "Подписаться"}</button>}
      </section>
      <div className="split">
        <aside className="card profile-card">
          <div className="profile-avatar">{initials(profile.nickname || profile.id)}</div>
          <h2>{profile.nickname || "-"}</h2>
          <div className="metric-row">
            <div className="metric"><strong>{toCount(profile.followersCount)}</strong><span>подписчиков</span></div>
            <div className="metric"><strong>{toCount(profile.followingCount)}</strong><span>подписок</span></div>
            <div className="metric"><strong>{toCount(polls.length)}</strong><span>опросов</span></div>
          </div>
        </aside>
        <section className="stack">
          <PollList items={polls} />
        </section>
      </div>
    </div>
  );
}
