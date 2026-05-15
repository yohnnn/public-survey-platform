import { FormEvent, useEffect, useMemo, useState } from "react";
import { Link, useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { ErrorState } from "../components/ErrorState";
import { LoadingState } from "../components/LoadingState";
import { PollList } from "../components/PollCard";
import { useToast } from "../components/Toast";
import type { FeedItem, PageMeta } from "../types/domain";
import { buildListSearch } from "../utils/format";

const feedConfig: Record<string, { title: string; subtitle: string; path: string; tags?: boolean; auth?: boolean }> = {
  "/feed": { title: "Лента", subtitle: "Новые публичные опросы", path: "/v1/feed", tags: true },
  "/trending": { title: "Тренды", subtitle: "Опросы с активным голосованием", path: "/v1/feed/trending" },
  "/following": { title: "Подписки", subtitle: "Опросы авторов, на которых вы подписаны", path: "/v1/feed/following", auth: true },
};

export function FeedPage() {
  const location = useLocation();
  const navigate = useNavigate();
  const toast = useToast();
  const { api, isAuthenticated } = useAuth();
  const [params] = useSearchParams();
  const config = feedConfig[location.pathname] || feedConfig["/feed"];
  const [items, setItems] = useState<FeedItem[]>([]);
  const [page, setPage] = useState<PageMeta | undefined>();
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState("");
  const search = useMemo(() => buildListSearch({ limit: params.get("limit") || "20", tags: params.get("tags") || "", includeTags: config.tags }), [config.tags, params]);

  useEffect(() => {
    let active = true;
    setStatus("loading");
    api
      .feed(config.path, search, config.auth)
      .then((response) => {
        if (!active) return;
        setItems(response.items || []);
        setPage(response.page);
        setStatus("ready");
      })
      .catch((err: unknown) => {
        if (!active) return;
        setError(err instanceof Error ? err.message : "Не удалось загрузить ленту.");
        setStatus("error");
      });
    return () => {
      active = false;
    };
  }, [api, config.auth, config.path, search]);

  if (config.auth && !isAuthenticated) {
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

  async function loadMore() {
    if (!page?.nextCursor) return;
    try {
      const moreSearch = buildListSearch({
        cursor: page.nextCursor,
        limit: String(page.limit || params.get("limit") || "20"),
        tags: params.get("tags") || "",
        includeTags: config.tags,
      });
      const response = await api.feed(config.path, moreSearch, config.auth);
      setItems((current) => [...current, ...(response.items || [])]);
      setPage(response.page);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Не удалось загрузить данные.", "error");
    }
  }

  function submitFilters(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const next = new URLSearchParams();
    const tags = String(form.get("tags") || "").trim();
    const limit = String(form.get("limit") || "20");
    if (tags) next.set("tags", tags);
    if (limit) next.set("limit", limit);
    navigate({ pathname: "/feed", search: next.toString() });
  }

  if (status === "loading") return <LoadingState title={config.title} />;
  if (status === "error") return <ErrorState message={error} onRetry={() => navigate(0)} />;

  return (
    <div className="page-view stack">
      <section className="page-head">
        <div>
          <h1>{config.title}</h1>
          <p>{config.subtitle}</p>
        </div>
        <Link className="button" to={isAuthenticated ? "/create" : "/auth"}>
          {isAuthenticated ? "Создать опрос" : "Войти"}
        </Link>
      </section>
      {config.tags ? (
        <section className="card">
          <form className="toolbar" onSubmit={submitFilters}>
            <label>
              Теги
              <input name="tags" defaultValue={params.get("tags") || ""} placeholder="например: sport, news" />
            </label>
            <label>
              Показать
              <select name="limit" defaultValue={params.get("limit") || "20"}>
                <option value="10">10</option>
                <option value="20">20</option>
                <option value="50">50</option>
              </select>
            </label>
            <button type="submit">Фильтровать</button>
            <Link className="button secondary" to="/feed">
              Сбросить
            </Link>
          </form>
        </section>
      ) : null}
      <PollList items={items} />
      {page?.hasMore && page.nextCursor ? (
        <div className="actions centered">
          <button className="secondary" type="button" onClick={loadMore}>
            Показать ещё
          </button>
        </div>
      ) : null}
    </div>
  );
}
