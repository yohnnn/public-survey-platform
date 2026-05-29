import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { ErrorState } from "../components/ErrorState";
import { LoadingState } from "../components/LoadingState";
import { PollList } from "../components/PollCard";
import { useToast } from "../components/Toast";
import { useLiveUpdates } from "../data/liveUpdates";
import type { FeedItem, PublicUserProfile } from "../types/domain";
import { initials, toCount } from "../utils/format";

export function ProfilePage() {
  const { id = "" } = useParams();
  const { api, isAuthenticated, me } = useAuth();
  const toast = useToast();
  const { mergeFeedItem, reconcilePolls } = useLiveUpdates();
  const [profile, setProfile] = useState<PublicUserProfile | null>(null);
  const [polls, setPolls] = useState<FeedItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [followBusy, setFollowBusy] = useState(false);

  useEffect(() => {
    let active = true;
    setLoading(true);
    Promise.all([api.profile(id, isAuthenticated), api.userPolls(id, "limit=20")])
      .then(([profileResponse, pollsResponse]) => {
        if (!active) return;
        const fetched = pollsResponse.items || [];
        reconcilePolls(fetched);
        setProfile(profileResponse.profile);
        setPolls(fetched.map(mergeFeedItem));
        setError("");
      })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Не удалось загрузить профиль."))
      .finally(() => active && setLoading(false));
    return () => {
      active = false;
    };
  }, [api, id, isAuthenticated, mergeFeedItem, reconcilePolls]);

  async function follow() {
    if (!profile || followBusy) return;
    const nextFollowing = !profile.isFollowing;
    setFollowBusy(true);
    try {
      await api.follow(profile.id, profile.isFollowing);
      const { profile: refreshed } = await api.profile(profile.id, true);
      setProfile(refreshed);
      toast(nextFollowing ? "Вы подписались." : "Вы отписались.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Не удалось изменить подписку.", "error");
    } finally {
      setFollowBusy(false);
    }
  }

  if (loading) return <LoadingState title="Профиль" />;
  if (error || !profile) return <ErrorState message={error} />;

  const isOwn = me?.id === profile.id;

  return (
    <div className="page-view stack profile-page">
      <section className="card profile-hero">
        <div className="profile-hero-main">
          <div className="profile-avatar">{initials(profile.nickname || profile.id)}</div>
          <div className="profile-hero-info stack">
            <h1>{profile.nickname || "Пользователь"}</h1>
            <div className="metric-row profile-metrics">
              <div className="metric">
                <strong>{toCount(profile.followersCount)}</strong>
                <span>подписчиков</span>
              </div>
              <div className="metric">
                <strong>{toCount(profile.followingCount)}</strong>
                <span>подписок</span>
              </div>
              <div className="metric">
                <strong>{toCount(polls.length)}</strong>
                <span>опросов</span>
              </div>
            </div>
          </div>
        </div>
        <div className="profile-hero-actions">
          {isOwn ? (
            <Link className="button secondary" to="/me/edit">
              Редактировать профиль
            </Link>
          ) : !isAuthenticated ? (
            <Link className="button" to="/auth">
              Войти, чтобы подписаться
            </Link>
          ) : (
            <button type="button" onClick={follow} disabled={followBusy}>
              {followBusy ? "..." : profile.isFollowing ? "Отписаться" : "Подписаться"}
            </button>
          )}
        </div>
      </section>

      <section className="stack profile-feed">
        <h2>Опросы</h2>
        <PollList items={polls} />
      </section>
    </div>
  );
}
