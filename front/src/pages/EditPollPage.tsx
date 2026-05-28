import { FormEvent, useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { ErrorState } from "../components/ErrorState";
import { LoadingState } from "../components/LoadingState";
import { TagPicker } from "../components/TagPicker";
import { useToast } from "../components/Toast";
import type { Poll } from "../types/domain";
import { toCount } from "../utils/format";

export function EditPollPage() {
  const { id = "" } = useParams();
  const { api, me } = useAuth();
  const toast = useToast();
  const navigate = useNavigate();
  const [poll, setPoll] = useState<Poll | null>(null);
  const [question, setQuestion] = useState("");
  const [tags, setTags] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    async function load() {
      setLoading(true);
      setError("");
      try {
        const response = await api.poll(id);
        const loaded = response.poll;
        if (!active) return;
        if (me && loaded.creatorId !== me.id) {
          navigate(`/poll/${encodeURIComponent(id)}`, { replace: true });
          return;
        }
        setPoll(loaded);
        setQuestion(loaded.question);
        setTags(loaded.tags || []);
      } catch (err) {
        if (!active) return;
        setError(err instanceof Error ? err.message : "Не удалось загрузить опрос.");
      } finally {
        if (active) setLoading(false);
      }
    }
    void load();
    return () => {
      active = false;
    };
  }, [api, id, me, navigate]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!poll) return;

    setBusy(true);
    try {
      const form = new FormData(event.currentTarget);
      const file = form.get("image") instanceof File && (form.get("image") as File).size > 0 ? (form.get("image") as File) : null;
      const imageUrl = file ? await api.uploadImage(file) : undefined;
      await api.updatePoll(poll.id, {
        question: question.trim(),
        tags,
        ...(imageUrl ? { imageUrl } : {}),
      });
      toast("Опрос обновлён.");
      navigate(`/poll/${encodeURIComponent(poll.id)}`);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Не удалось обновить опрос.", "error");
    } finally {
      setBusy(false);
    }
  }

  async function deletePoll() {
    if (!poll || deleting || !window.confirm("Удалить опрос без восстановления?")) return;
    setDeleting(true);
    try {
      await api.deletePoll(poll.id);
      toast("Опрос удалён.");
      navigate(me?.id ? `/profile/${encodeURIComponent(me.id)}` : "/me");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Не удалось удалить опрос.", "error");
    } finally {
      setDeleting(false);
    }
  }

  if (loading) return <LoadingState title="Редактирование" />;
  if (error || !poll) return <ErrorState message={error || "Опрос не найден."} onRetry={() => window.location.reload()} />;

  return (
    <div className="page-view stack">
      <section className="page-head">
        <div>
          <h1>Редактировать опрос</h1>
          <p>Измените вопрос, теги или изображение.</p>
        </div>
        <Link className="button secondary" to={`/poll/${encodeURIComponent(poll.id)}`}>
          Отмена
        </Link>
      </section>

      <section className="card form-page stack">
        <form className="stack" onSubmit={handleSubmit}>
          <label>
            Вопрос
            <textarea
              name="question"
              maxLength={300}
              required
              value={question}
              onChange={(event) => setQuestion(event.target.value)}
            />
          </label>

          <TagPicker value={tags} onChange={setTags} />

          <div className="stack">
            <span className="option-editor-label">Варианты ответа</span>
            <p className="hint small">Варианты нельзя изменить после публикации.</p>
            <div className="option-list">
              {(poll.options || []).map((option) => (
                <div className="option-editor-row readonly" key={option.id}>
                  <input type="text" value={option.text} readOnly />
                  <span className="chip">{toCount(option.votesCount)} голосов</span>
                </div>
              ))}
            </div>
          </div>

          {poll.imageUrl ? (
            <div className="stack">
              <span className="option-editor-label">Текущее изображение</span>
              <img className="poll-image" src={poll.imageUrl} alt="" />
            </div>
          ) : null}

          <label>
            {poll.imageUrl ? "Заменить изображение" : "Изображение"}
            <input name="image" type="file" accept="image/*" />
          </label>

          <div className="actions">
            <button type="submit" disabled={busy}>
              {busy ? "Сохраняю..." : "Сохранить"}
            </button>
            <button className="danger" type="button" disabled={deleting || busy} onClick={deletePoll}>
              {deleting ? "Удаление..." : "Удалить опрос"}
            </button>
          </div>
        </form>
      </section>
    </div>
  );
}
