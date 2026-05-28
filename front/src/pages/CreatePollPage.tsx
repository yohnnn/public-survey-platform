import { FormEvent, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { PollOptionsEditor } from "../components/PollOptionsEditor";
import { TagPicker } from "../components/TagPicker";
import { useToast } from "../components/Toast";

export function CreatePollPage() {
  const { api } = useAuth();
  const toast = useToast();
  const navigate = useNavigate();
  const [busy, setBusy] = useState(false);
  const [question, setQuestion] = useState("");
  const [pollType, setPollType] = useState("POLL_TYPE_SINGLE_CHOICE");
  const [options, setOptions] = useState(["", ""]);
  const [selectedTags, setSelectedTags] = useState<string[]>([]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalizedOptions = options.map((item) => item.trim()).filter(Boolean);

    if (normalizedOptions.length < 2) {
      toast("Добавьте минимум два варианта ответа.", "error");
      return;
    }

    setBusy(true);
    try {
      const form = event.currentTarget;
      const data = new FormData(form);
      const file = data.get("image") instanceof File && (data.get("image") as File).size > 0 ? (data.get("image") as File) : null;
      const imageUrl = file ? await api.uploadImage(file) : "";
      const response = await api.createPoll({
        question: question.trim(),
        type: pollType,
        options: normalizedOptions,
        tags: selectedTags,
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

  return (
    <div className="page-view stack">
      <section className="page-head">
        <div>
          <h1>Создать опрос</h1>
          <p>Задайте вопрос и добавьте варианты ответа.</p>
        </div>
      </section>

      <section className="card form-page">
        <form className="stack" onSubmit={handleSubmit}>
          <label>
            Вопрос
            <textarea
              name="question"
              maxLength={300}
              required
              placeholder="О чём спросим аудиторию?"
              value={question}
              onChange={(event) => setQuestion(event.target.value)}
            />
          </label>

          <label>
            Тип голосования
            <select name="type" value={pollType} onChange={(event) => setPollType(event.target.value)}>
              <option value="POLL_TYPE_SINGLE_CHOICE">Один вариант</option>
              <option value="POLL_TYPE_MULTIPLE_CHOICE">Несколько вариантов</option>
            </select>
          </label>

          <PollOptionsEditor value={options} onChange={setOptions} />
          <TagPicker value={selectedTags} onChange={setSelectedTags} />

          <label>
            Изображение
            <input name="image" type="file" accept="image/*" />
          </label>

          <button type="submit" disabled={busy}>
            {busy ? "Публикую..." : "Опубликовать"}
          </button>
        </form>
      </section>
    </div>
  );
}
