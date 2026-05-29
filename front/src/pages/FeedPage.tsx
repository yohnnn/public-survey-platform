import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { FeedFilters } from "../components/FeedFilters";
import { ErrorState } from "../components/ErrorState";
import { LoadingState } from "../components/LoadingState";
import { PollList } from "../components/PollCard";
import { feedModeByPath } from "../config/feed";
import { useLiveUpdates } from "../data/liveUpdates";
import type { FeedItem, PageMeta } from "../types/domain";
import { buildListSearch } from "../utils/format";

const FEED_LIMIT = "30";

const modeCopy: Record<string, { lead: string }> = {
  "/": { lead: "Новые опросы получают минимальный охват, затем — общая лента" },
  "/trending": { lead: "Самые обсуждаемые опросы прямо сейчас" },
  "/following": { lead: "Публикации авторов, на которых вы подписаны" },
};

export function FeedPage() {
  const location = useLocation();
  const navigate = useNavigate();
  const { api, isAuthenticated } = useAuth();
  const { mergeFeedItem, reconcilePolls } = useLiveUpdates();
  const [params, setSearchParams] = useSearchParams();
  const [items, setItems] = useState<FeedItem[]>([]);
  const [page, setPage] = useState<PageMeta | undefined>();
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [loadingMore, setLoadingMore] = useState(false);
  const [filtersOpen, setFiltersOpen] = useState(false);
  const [error, setError] = useState("");
  const sentinelRef = useRef<HTMLDivElement | null>(null);
  const impressionQueueRef = useRef<Set<string>>(new Set());
  const impressionFlushRef = useRef<number | null>(null);
  const recordedImpressionsRef = useRef<Set<string>>(new Set());

  const mode = feedModeByPath(location.pathname);
  const activeTags = useMemo(
    () => (mode.tags ? params.getAll("tags").filter(Boolean) : []),
    [mode.tags, params],
  );
  const copy = modeCopy[mode.path] || modeCopy["/"];

  const initialSearch = useMemo(
    () => buildListSearch({ limit: FEED_LIMIT, tags: activeTags, includeTags: mode.tags }),
    [activeTags, mode.tags],
  );

  const flushImpressions = useCallback(() => {
    if (mode.path !== "/" || impressionQueueRef.current.size === 0) return;
    const feedItemIds = Array.from(impressionQueueRef.current);
    impressionQueueRef.current.clear();
    feedItemIds.forEach((id) => recordedImpressionsRef.current.add(id));
    void api.recordFeedImpressions(feedItemIds).catch(() => {
      feedItemIds.forEach((id) => recordedImpressionsRef.current.delete(id));
    });
  }, [api, mode.path]);

  const queueImpression = useCallback(
    (feedItemId: string) => {
      if (mode.path !== "/" || !feedItemId || recordedImpressionsRef.current.has(feedItemId)) return;
      impressionQueueRef.current.add(feedItemId);
      if (impressionFlushRef.current !== null) {
        window.clearTimeout(impressionFlushRef.current);
      }
      impressionFlushRef.current = window.setTimeout(() => {
        impressionFlushRef.current = null;
        flushImpressions();
      }, 400);
    },
    [flushImpressions, mode.path],
  );

  useEffect(() => {
    return () => {
      if (impressionFlushRef.current !== null) {
        window.clearTimeout(impressionFlushRef.current);
      }
      flushImpressions();
    };
  }, [flushImpressions]);

  useEffect(() => {
    recordedImpressionsRef.current.clear();
    impressionQueueRef.current.clear();
  }, [location.pathname, initialSearch]);

  const loadFeed = useCallback(
    async (search: string, append = false) => {
      if (append) setLoadingMore(true);
      else setStatus("loading");

      try {
        const response = await api.feed(mode.apiPath, search, mode.auth);
        const fetched = response.items || [];
        reconcilePolls(fetched);
        setItems((current) => (append ? [...current, ...fetched.map(mergeFeedItem)] : fetched.map(mergeFeedItem)));
        setPage(response.page);
        setError("");
        setStatus("ready");
      } catch (err) {
        const message = err instanceof Error ? err.message : "Не удалось загрузить ленту.";
        if (append) setError(message);
        else {
          setError(message);
          setStatus("error");
        }
      } finally {
        setLoadingMore(false);
      }
    },
    [api, mergeFeedItem, mode.apiPath, mode.auth, reconcilePolls],
  );

  useEffect(() => {
    setFiltersOpen(false);
    void loadFeed(initialSearch);
  }, [initialSearch, loadFeed, location.pathname]);

  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel || status !== "ready") return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (!entries[0]?.isIntersecting || loadingMore || !page?.hasMore || !page.nextCursor) return;
        const moreSearch = buildListSearch({
          cursor: page.nextCursor,
          limit: FEED_LIMIT,
          tags: activeTags,
          includeTags: mode.tags,
        });
        void loadFeed(moreSearch, true);
      },
      { rootMargin: "240px" },
    );

    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [activeTags, loadFeed, loadingMore, mode.tags, page, status]);

  function toggleTag(tagId: string) {
    const next = new URLSearchParams(params);
    const current = next.getAll("tags");
    next.delete("tags");
    if (current.includes(tagId)) {
      current.filter((tag) => tag !== tagId).forEach((tag) => next.append("tags", tag));
    } else {
      [...current, tagId].forEach((tag) => next.append("tags", tag));
    }
    setSearchParams(next);
  }

  function removeTag(tagId: string) {
    const next = new URLSearchParams(params);
    const current = next.getAll("tags").filter((tag) => tag !== tagId);
    next.delete("tags");
    current.forEach((tag) => next.append("tags", tag));
    setSearchParams(next);
  }

  function clearTags() {
    setSearchParams({});
  }

  if (mode.auth && !isAuthenticated) {
    return (
      <div className="feed-empty-state">
        <h1>Подписки</h1>
        <p>Войдите, чтобы видеть опросы авторов, на которых вы подписаны.</p>
        <Link className="button" to="/auth">
          Войти
        </Link>
      </div>
    );
  }

  if (status === "loading" && items.length === 0) return <LoadingState title={mode.label} />;
  if (status === "error" && items.length === 0) return <ErrorState message={error} onRetry={() => navigate(0)} />;

  return (
    <div className="feed-view">
      <header className="feed-header">
        <div>
          <h1>{mode.label}</h1>
          <p className="feed-lead">{copy.lead}</p>
        </div>
      </header>

      {mode.tags ? (
        <FeedFilters
          activeTags={activeTags}
          open={filtersOpen}
          onToggle={() => setFiltersOpen((value) => !value)}
          onSelect={toggleTag}
          onRemove={removeTag}
          onClear={clearTags}
        />
      ) : null}

      <PollList items={items} onPollVisible={mode.path === "/" ? queueImpression : undefined} />
      <div ref={sentinelRef} className="feed-sentinel" aria-hidden="true" />
      {loadingMore ? <p className="feed-loading-more">Загружаем ещё...</p> : null}
    </div>
  );
}
