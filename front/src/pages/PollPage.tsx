import { FormEvent, useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { ErrorState } from "../components/ErrorState";
import { LoadingState } from "../components/LoadingState";
import { ResultRow } from "../components/PollCard";
import { useToast } from "../components/Toast";
import type { AgeStat, CountryStat, GenderStat, Poll, PollAnalytics, PublicUserProfile, VoteState } from "../types/domain";
import { formatDate, splitCSV, toCount } from "../utils/format";

export function PollPage() {
  const { id = "" } = useParams();
  const { api, me, isAuthenticated } = useAuth();
  const toast = useToast();
  const navigate = useNavigate();
  const [state, setState] = useState<{
    poll?: Poll;
    author?: PublicUserProfile;
    vote?: VoteState;
    analytics?: PollAnalytics;
    countries: CountryStat[];
    gender: GenderStat[];
    age: AgeStat[];
    loading: boolean;
    error: string;
    tab: "results" | "analytics";
  }>({ countries: [], gender: [], age: [], loading: true, error: "", tab: "results" });

  useEffect(() => {
    let active = true;
    async function load() {
      setState((current) => ({ ...current, loading: true, error: "" }));
      try {
        const pollResponse = await api.poll(id);
        const poll = pollResponse.poll;
        const [author, analytics, countries, gender, age, vote] = await Promise.allSettled([
          api.profile(poll.creatorId, isAuthenticated),
          api.analytics(poll.id),
          api.analyticsSection<CountryStat>(poll.id, "countries"),
          api.analyticsSection<GenderStat>(poll.id, "gender"),
          api.analyticsSection<AgeStat>(poll.id, "age"),
          isAuthenticated ? api.userVote(poll.id) : Promise.resolve({ hasVoted: false, optionIds: [], pollId: poll.id }),
        ]);
        if (!active) return;
        setState({
          poll,
          author: settled(author)?.profile,
          analytics: settled(analytics),
          countries: settled(countries)?.items || [],
          gender: settled(gender)?.items || [],
          age: settled(age)?.items || [],
          vote: settled(vote),
          loading: false,
          error: "",
          tab: "results",
        });
      } catch (error) {
        if (!active) return;
        setState((current) => ({ ...current, loading: false, error: error instanceof Error ? error.message : "Не удалось загрузить опрос." }));
      }
    }
    void load();
    return () => {
      active = false;
    };
  }, [api, id, isAuthenticated]);

  const poll = state.poll;
  const selected = useMemo(() => new Set(state.vote?.optionIds || []), [state.vote]);

  async function reloadPoll() {
    navigate(0);
  }

  async function submitVote(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!poll) return;
    const optionIds = [...event.currentTarget.querySelectorAll<HTMLInputElement>("input[name='optionId']:checked")].map((input) => input.value);
    if (!optionIds.length) {
      toast("Выберите вариант.", "error");
      return;
    }
    try {
      await api.vote(poll.id, optionIds);
      toast("Голос сохранён.");
      reloadPoll();
    } catch (error) {
      toast(error instanceof Error ? error.message : "Не удалось сохранить голос.", "error");
    }
  }

  async function removeVote() {
    if (!poll) return;
    try {
      await api.removeVote(poll.id);
      toast("Голос удалён.");
      reloadPoll();
    } catch (error) {
      toast(error instanceof Error ? error.message : "Не удалось удалить голос.", "error");
    }
  }

  async function updatePoll(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!poll) return;
    const form = new FormData(event.currentTarget);
    try {
      const file = form.get("image") instanceof File && (form.get("image") as File).size > 0 ? (form.get("image") as File) : null;
      const imageUrl = file ? await api.uploadImage(file) : undefined;
      await api.updatePoll(poll.id, {
        question: String(form.get("question") || "").trim(),
        tags: splitCSV(String(form.get("tags") || "")),
        ...(imageUrl ? { imageUrl } : {}),
      });
      toast("Опрос обновлён.");
      reloadPoll();
    } catch (error) {
      toast(error instanceof Error ? error.message : "Не удалось обновить опрос.", "error");
    }
  }

  async function deletePoll() {
    if (!poll || !window.confirm("Удалить опрос без восстановления?")) return;
    try {
      await api.deletePoll(poll.id);
      toast("Опрос удалён.");
      navigate("/me");
    } catch (error) {
      toast(error instanceof Error ? error.message : "Не удалось удалить опрос.", "error");
    }
  }

  if (state.loading) return <LoadingState title="Опрос" />;
  if (state.error || !poll) return <ErrorState message={state.error} onRetry={reloadPoll} />;

  const isOwner = me?.id === poll.creatorId;
  const isMultiple = poll.type === "POLL_TYPE_MULTIPLE_CHOICE";
  const inputType = isMultiple ? "checkbox" : "radio";

  return (
    <div className="page-view stack">
      <section className="page-head">
        <div>
          <h1>{poll.question}</h1>
          <p>
            Автор: <Link to={`/profile/${encodeURIComponent(poll.creatorId)}`}>{state.author?.nickname || poll.creatorId}</Link>
          </p>
        </div>
        <span className="chip">{formatDate(poll.createdAt)}</span>
      </section>

      <div className="split">
        <section className="card stack">
          {poll.imageUrl ? <img className="poll-image" src={poll.imageUrl} alt="" /> : null}
          <div className="tag-list">
            {(poll.tags || []).map((tag) => (
              <Link key={tag} className="tag" to={`/feed?tags=${encodeURIComponent(tag)}`}>
                {tag}
              </Link>
            ))}
          </div>
          <div className="metric-row">
            <div className="metric"><strong>{toCount(poll.totalVotes)}</strong><span>голосов</span></div>
            <div className="metric"><strong>{poll.options?.length || 0}</strong><span>вариантов</span></div>
            <div className="metric"><strong>{isMultiple ? "multi" : "single"}</strong><span>тип</span></div>
          </div>

          {isAuthenticated ? (
            <form className="stack" onSubmit={submitVote}>
              <h3>{state.vote?.hasVoted ? "Ваш голос" : "Ваш выбор"}</h3>
              <div className="option-list">
                {(poll.options || []).map((option) => (
                  <label className="vote-option" key={option.id}>
                    <input type={inputType} name="optionId" value={option.id} defaultChecked={selected.has(option.id)} />
                    <span>{option.text}</span>
                    <strong>{toCount(option.votesCount)}</strong>
                  </label>
                ))}
              </div>
              <div className="actions">
                <button type="submit">{state.vote?.hasVoted ? "Изменить голос" : "Проголосовать"}</button>
                <button className="secondary" type="button" disabled={!state.vote?.hasVoted} onClick={removeVote}>
                  Убрать голос
                </button>
              </div>
            </form>
          ) : (
            <div className="empty">Войдите, чтобы проголосовать. <Link to="/auth">Авторизация</Link></div>
          )}

          {isOwner ? (
            <section className="card nested-card stack">
              <h3>Управление опросом</h3>
              <form className="stack" onSubmit={updatePoll}>
                <label>Вопрос<textarea name="question" required defaultValue={poll.question} /></label>
                <label>Теги<input name="tags" defaultValue={(poll.tags || []).join(", ")} /></label>
                <label>Заменить изображение<input name="image" type="file" accept="image/*" /></label>
                <div className="actions">
                  <button type="submit">Сохранить</button>
                  <button className="danger" type="button" onClick={deletePoll}>Удалить</button>
                </div>
              </form>
            </section>
          ) : null}
        </section>

        <aside className="card stack">
          <div className="tabbar compact-tabs">
            <button className={state.tab === "results" ? "active" : ""} type="button" onClick={() => setState((current) => ({ ...current, tab: "results" }))}>Результаты</button>
            <button className={state.tab === "analytics" ? "active" : ""} type="button" onClick={() => setState((current) => ({ ...current, tab: "analytics" }))}>Аналитика</button>
          </div>
          {state.tab === "results" ? (
            <section>
              <h3>Результаты</h3>
              <div className="option-list">{(poll.options || []).map((option) => <ResultRow key={option.id} label={option.text} votes={option.votesCount} totalVotes={poll.totalVotes} />)}</div>
            </section>
          ) : (
            <AnalyticsPanel poll={poll} analytics={state.analytics} countries={state.countries} gender={state.gender} age={state.age} />
          )}
        </aside>
      </div>
    </div>
  );
}

function AnalyticsPanel({ poll, analytics, countries, gender, age }: { poll: Poll; analytics?: PollAnalytics; countries: CountryStat[]; gender: GenderStat[]; age: AgeStat[] }) {
  const optionNames = new Map((poll.options || []).map((option) => [option.id, option.text]));
  return (
    <section className="stack">
      <h3>Аналитика</h3>
      <div className="stats-list">
        <div className="stats-item"><span>Всего</span><strong>{toCount(analytics?.totalVotes)}</strong></div>
        {(analytics?.options || []).map((item) => (
          <div className="stats-item" key={item.optionId}><span>{optionNames.get(item.optionId) || item.optionId}</span><strong>{toCount(item.votes)}</strong></div>
        ))}
      </div>
      <Stats title="Страны" items={countries} label={(item) => item.country} />
      <Stats title="Пол" items={gender} label={(item) => item.gender} />
      <Stats title="Возраст" items={age} label={(item) => item.ageRange} />
    </section>
  );
}

function Stats<T extends { votes: number }>({ title, items, label }: { title: string; items: T[]; label: (item: T) => string }) {
  return (
    <>
      <h4>{title}</h4>
      {items.length ? <div className="stats-list">{items.map((item) => <div className="stats-item" key={label(item) || String(item.votes)}><span>{label(item) || "-"}</span><strong>{toCount(item.votes)}</strong></div>)}</div> : <div className="empty">Данных пока нет.</div>}
    </>
  );
}

function settled<T>(result: PromiseSettledResult<T>): T | undefined {
  return result.status === "fulfilled" ? result.value : undefined;
}
