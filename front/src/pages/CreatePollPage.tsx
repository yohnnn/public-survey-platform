import { FormEvent, useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { LoadingState } from "../components/LoadingState";
import { ResultRow } from "../components/PollCard";
import { useToast } from "../components/Toast";
import type { Tag } from "../types/domain";
import { splitCSV } from "../utils/format";

export function CreatePollPage() {
  const { api } = useAuth();
  const toast = useToast();
  const navigate = useNavigate();
  const [tags, setTags] = useState<Tag[]>([]);
  const [busy, setBusy] = useState(false);
  const [preview, setPreview] = useState({ question: "", options: ["Первый вариант", "Второй вариант"], tags: [] as string[] });

  useEffect(() => {
    api
      .tags()
      .then((response) => setTags(response.items || []))
      .catch(() => undefined);
  }, [api]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    const options = String(data.get("options") || "")
      .split("\n")
      .map((item) => item.trim())
      .filter(Boolean);

    if (options.length < 2) {
      toast("Добавьте минимум два варианта ответа.", "error");
      return;
    }

    setBusy(true);
    try {
      const file = data.get("image") instanceof File && (data.get("image") as File).size > 0 ? (data.get("image") as File) : null;
      const imageUrl = file ? await api.uploadImage(file) : "";
      const response = await api.createPoll({
        question: String(data.get("question") || "").trim(),
        type: String(data.get("type") || "POLL_TYPE_SINGLE_CHOICE"),
        options,
        tags: splitCSV(String(data.get("tags") || "")),
        imageUrl,
      });
      toast("Опрос опубликован.");
      navigate(`/poll/${encodeURIComponent(response.poll.id)}`);
    } catch (error) {
      toast(error instanceof Error ? error.message : "Не удалось создать опрос.", "error");
    } finally {
      setBusy(false);
    }
  }

  function updatePreview(form: HTMLFormElement) {
    const data = new FormData(form);
    const options = String(data.get("options") || "")
      .split("\n")
      .map((item) => item.trim())
      .filter(Boolean);
    setPreview({
      question: String(data.get("question") || ""),
      options: options.length ? options : ["Первый вариант", "Второй вариант"],
      tags: splitCSV(String(data.get("tags") || "")),
    });
  }

  function appendTag(form: HTMLFormElement, tag: string) {
    const input = form.elements.namedItem("tags") as HTMLInputElement | null;
    if (!input) return;
    const current = splitCSV(input.value);
    if (!current.includes(tag)) current.push(tag);
    input.value = current.join(", ");
    updatePreview(form);
  }

  if (!tags) return <LoadingState title="Новый опрос" />;

  return (
    <div className="page-view stack">
      <section className="page-head">
        <div>
          <h1>Создать опрос</h1>
          <p>Добавьте вопрос, варианты ответа и при необходимости изображение.</p>
        </div>
      </section>
      <div className="split">
        <section className="card">
          <form id="create-poll-form" className="stack" onSubmit={handleSubmit} onInput={(event) => updatePreview(event.currentTarget)}>
            <label>
              Вопрос
              <textarea name="question" maxLength={300} required placeholder="О чём спросим аудиторию?" />
            </label>
            <label>
              Тип голосования
              <select name="type" defaultValue="POLL_TYPE_SINGLE_CHOICE">
                <option value="POLL_TYPE_SINGLE_CHOICE">Один вариант</option>
                <option value="POLL_TYPE_MULTIPLE_CHOICE">Несколько вариантов</option>
              </select>
            </label>
            <label>
              Варианты ответа
              <textarea name="options" required placeholder={"Да\nНет\nПока не знаю"} />
            </label>
            <label>
              Теги
              <input name="tags" placeholder="news, sport, city" />
            </label>
            <label>
              Изображение
              <input name="image" type="file" accept="image/*" />
            </label>
            <button type="submit" disabled={busy}>
              {busy ? "Публикую..." : "Опубликовать"}
            </button>
          </form>
        </section>
        <aside className="card stack sticky-panel">
          <div>
            <h3>Предпросмотр</h3>
            <article className="mini-poll">
              <h4>{preview.question || "Ваш вопрос появится здесь"}</h4>
              <div className="option-list">
                {preview.options.slice(0, 4).map((option) => (
                  <ResultRow key={option} label={option} votes={0} totalVotes={0} />
                ))}
              </div>
              <div className="tag-list">{preview.tags.length ? preview.tags.map((tag) => <span className="tag" key={tag}>{tag}</span>) : <span className="chip">без тегов</span>}</div>
            </article>
          </div>
          <h3>Популярные теги</h3>
          <div className="tag-list">
            {tags.length ? (
              tags.map((tag) => (
                <button key={tag.id} className="secondary tag-pick" type="button" onClick={() => {
                  const form = document.getElementById("create-poll-form") as HTMLFormElement | null;
                  if (form) appendTag(form, tag.name);
                }}>
                  {tag.name}
                </button>
              ))
            ) : (
              <span className="muted">Тегов пока нет.</span>
            )}
          </div>
          <p className="hint small">Теги помогают пользователям находить опросы в ленте.</p>
          <Link className="button secondary" to="/tags">
            Управлять тегами
          </Link>
        </aside>
      </div>
    </div>
  );
}
