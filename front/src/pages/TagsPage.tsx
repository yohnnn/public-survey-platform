import { FormEvent, useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { LoadingState } from "../components/LoadingState";
import { useToast } from "../components/Toast";
import type { Tag } from "../types/domain";

export function TagsPage() {
  const { api, isAuthenticated } = useAuth();
  const toast = useToast();
  const [tags, setTags] = useState<Tag[]>([]);
  const [loading, setLoading] = useState(true);

  const loadTags = useCallback(async () => {
    setLoading(true);
    try {
      const response = await api.tags();
      setTags(response.items || []);
    } finally {
      setLoading(false);
    }
  }, [api]);

  useEffect(() => {
    void loadTags();
  }, [loadTags]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    try {
      await api.createTag(String(data.get("name") || "").trim());
      event.currentTarget.reset();
      toast("Тег создан.");
      await loadTags();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Не удалось создать тег.", "error");
    }
  }

  if (loading) return <LoadingState title="Теги" />;

  return (
    <div className="page-view stack">
      <section className="page-head">
        <div>
          <h1>Теги</h1>
          <p>Используйте теги, чтобы группировать опросы.</p>
        </div>
      </section>
      <div className="grid-two">
        <section className="card stack">
          <h2>Все теги</h2>
          <div className="tag-list">
            {tags.length ? tags.map((tag) => <Link key={tag.id} className="tag" to={`/feed?tags=${encodeURIComponent(tag.name)}`}>{tag.name}</Link>) : <span className="muted">Тегов пока нет.</span>}
          </div>
        </section>
        <section className="card stack">
          <h2>Новый тег</h2>
          {isAuthenticated ? (
            <form className="stack" onSubmit={submit}>
              <label>Название<input name="name" required /></label>
              <button type="submit">Создать тег</button>
            </form>
          ) : (
            <div className="empty">Войдите, чтобы создавать теги.</div>
          )}
        </section>
      </div>
    </div>
  );
}
