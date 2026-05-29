import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { ErrorState } from "../components/ErrorState";
import { LoadingState } from "../components/LoadingState";
import { PollAnalyticsCharts } from "../components/PollAnalyticsCharts";
import { useToast } from "../components/Toast";
import { pollToLive, useLiveUpdates } from "../data/liveUpdates";
import { tagLabel } from "../data/tags";
import type { AgeStat, CountryStat, GenderStat, Poll, PollAnalytics, PublicUserProfile, VoteState } from "../types/domain";
import { detectAgeRange, formatDate, normalizeGender, toCount } from "../utils/format";

export function PollPage() {
  const { id = "" } = useParams();
  const { api, me, isAuthenticated } = useAuth();
  const { mergePoll, setPollLive } = useLiveUpdates();
  const toast = useToast();
  const [analyticsKey, setAnalyticsKey] = useState(0);
  const [voteBusy, setVoteBusy] = useState(false);
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
  }>({ countries: [], gender: [], age: [], loading: true, error: "" });

  useEffect(() => {
    let active = true;
    async function load() {
      setState((current) => ({ ...current, loading: true, error: "" }));
      try {
        const pollResponse = await api.poll(id);
        const poll = pollResponse.poll;
        const [author, vote] = await Promise.allSettled([
          api.profile(poll.creatorId, isAuthenticated),
          isAuthenticated ? api.userVote(poll.id) : Promise.resolve({ hasVoted: false, optionIds: [], pollId: poll.id }),
        ]);
        if (!active) return;
        setState((current) => ({
          ...current,
          poll: mergePoll(poll),
          author: settled(author)?.profile,
          vote: settled(vote),
          loading: false,
          error: "",
        }));
      } catch (error) {
        if (!active) return;
        setState((current) => ({ ...current, loading: false, error: error instanceof Error ? error.message : "Не удалось загрузить опрос." }));
      }
    }
    void load();
    return () => {
      active = false;
    };
  }, [api, id, isAuthenticated, mergePoll]);

  const loadAnalytics = useCallback(async () => {
    if (!id) return;
    try {
      const [analytics, countries, gender, age] = await Promise.allSettled([
        api.analytics(id),
        api.analyticsSection<CountryStat>(id, "countries"),
        api.analyticsSection<GenderStat>(id, "gender"),
        api.analyticsSection<AgeStat>(id, "age"),
      ]);
      setState((current) => ({
        ...current,
        analytics: settled(analytics),
        countries: settled(countries)?.items || [],
        gender: settled(gender)?.items || [],
        age: settled(age)?.items || [],
      }));
      return {
        countries: settled(countries)?.items?.length || 0,
        gender: settled(gender)?.items?.length || 0,
        age: settled(age)?.items?.length || 0,
        totalVotes: Number(settled(analytics)?.totalVotes || 0),
      };
    } catch {
      return undefined;
    }
  }, [api, id]);

  useEffect(() => {
    void loadAnalytics();
  }, [loadAnalytics, analyticsKey]);

  const poll = state.poll;
  const selected = useMemo(() => new Set(state.vote?.optionIds || []), [state.vote]);

  function applyDemographicDelta(delta: number) {
    if (!delta || !me) return {};
    const countries = adjustStat(state.countries, "country", me.country, delta, (country) => ({ country, votes: delta }));
    const gender = adjustStat(state.gender, "gender", normalizeGender(me.gender), delta, (gender) => ({
      gender,
      votes: delta,
    }));
    const ageRange = detectAgeRange(me.birthYear);
    const age = ageRange
      ? adjustStat(state.age, "ageRange", ageRange, delta, (ageRange) => ({ ageRange, votes: delta }))
      : state.age;
    return { countries, gender, age };
  }

  function applyVoteOptimistic(optionIds: string[]) {
    if (!poll) return;
    const prevVote = state.vote;
    const prevOptions = poll.options;

    const oldIds = new Set(prevVote?.optionIds || []);
    const newIds = new Set(optionIds);

    const nextOptions = prevOptions.map((opt) => {
      let delta = 0;
      if (oldIds.has(opt.id) && !newIds.has(opt.id)) delta = -1;
      if (!oldIds.has(opt.id) && newIds.has(opt.id)) delta = 1;
      return delta ? { ...opt, votesCount: Number(opt.votesCount || 0) + delta } : opt;
    });

    const voteDelta = prevVote?.hasVoted ? 0 : 1;
    const nextTotalVotes = Number(poll.totalVotes || 0) + voteDelta;
    const demographicDelta = voteDelta ? applyDemographicDelta(1) : {};

    setState((current) => ({
      ...current,
      poll: { ...poll, options: nextOptions, totalVotes: nextTotalVotes },
      vote: { pollId: poll.id, hasVoted: true, optionIds },
      ...demographicDelta,
    }));
    setPollLive(poll.id, pollToLive({ options: nextOptions, totalVotes: nextTotalVotes }));
  }

  function revertVoteOptimistic(
    prevPoll: Poll,
    prevVote?: VoteState,
    prevAnalytics?: Pick<typeof state, "countries" | "gender" | "age">,
  ) {
    setState((current) => ({
      ...current,
      poll: prevPoll,
      vote: prevVote,
      ...(prevAnalytics || {}),
    }));
    setPollLive(prevPoll.id, pollToLive(prevPoll));
  }

  async function submitVote(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!poll) return;
    const optionIds = [...event.currentTarget.querySelectorAll<HTMLInputElement>("input[name='optionId']:checked")].map((input) => input.value);
    if (!optionIds.length) {
      toast("Выберите вариант.", "error");
      return;
    }

    const prevPoll = { ...poll, options: poll.options.map((o) => ({ ...o })) };
    const prevVote = state.vote ? { ...state.vote, optionIds: [...state.vote.optionIds] } : undefined;
    const prevAnalytics = { countries: [...state.countries], gender: [...state.gender], age: [...state.age] };

    applyVoteOptimistic(optionIds);
    setVoteBusy(true);
    try {
      await api.vote(poll.id, optionIds);
      toast("Голос сохранён.");
      scheduleAnalyticsRefresh(Number(poll.totalVotes || 0) + (prevVote?.hasVoted ? 0 : 1));
    } catch (error) {
      revertVoteOptimistic(prevPoll, prevVote, prevAnalytics);
      toast(error instanceof Error ? error.message : "Не удалось сохранить голос.", "error");
    } finally {
      setVoteBusy(false);
    }
  }

  async function removeVote() {
    if (!poll || voteBusy) return;

    const prevPoll = { ...poll, options: poll.options.map((o) => ({ ...o })) };
    const prevVote = state.vote ? { ...state.vote, optionIds: [...state.vote.optionIds] } : undefined;
    const prevAnalytics = { countries: [...state.countries], gender: [...state.gender], age: [...state.age] };

    const removedIds = new Set(state.vote?.optionIds || []);
    const nextOptions = poll.options.map((opt) =>
      removedIds.has(opt.id) ? { ...opt, votesCount: Math.max(0, Number(opt.votesCount || 0) - 1) } : opt,
    );
    const nextTotalVotes = Math.max(0, Number(poll.totalVotes || 0) - 1);

    setState((current) => ({
      ...current,
      poll: { ...poll, options: nextOptions, totalVotes: nextTotalVotes },
      vote: { pollId: poll.id, hasVoted: false, optionIds: [] },
      ...applyDemographicDelta(-1),
    }));
    setPollLive(poll.id, pollToLive({ options: nextOptions, totalVotes: nextTotalVotes }));

    setVoteBusy(true);
    try {
      await api.removeVote(poll.id);
      toast("Голос удалён.");
      scheduleAnalyticsRefresh(Math.max(0, Number(poll.totalVotes || 0) - 1));
    } catch (error) {
      revertVoteOptimistic(prevPoll, prevVote, prevAnalytics);
      toast(error instanceof Error ? error.message : "Не удалось удалить голос.", "error");
    } finally {
      setVoteBusy(false);
    }
  }

  function scheduleAnalyticsRefresh(expectedVotes: number) {
    let attempts = 0;

    const tick = async () => {
      attempts += 1;
      const snapshot = await loadAnalytics();
      const synced = snapshot && Number(snapshot.totalVotes || 0) >= expectedVotes;

      if (synced || attempts >= 10) {
        return;
      }
      window.setTimeout(() => void tick(), 500);
    };

    window.setTimeout(() => void tick(), 400);
  }

  if (state.loading) return <LoadingState title="Опрос" />;
  if (state.error || !poll) return <ErrorState message={state.error} onRetry={() => window.location.reload()} />;

  const isOwner = me?.id === poll.creatorId;
  const isMultiple = poll.type === "POLL_TYPE_MULTIPLE_CHOICE";
  const inputType = isMultiple ? "checkbox" : "radio";

  return (
    <div className="page-view stack poll-page">
      <section className="page-head">
        <div>
          <h1>{poll.question}</h1>
          <p className="muted small">
            <Link to={`/profile/${encodeURIComponent(poll.creatorId)}`}>{state.author?.nickname || poll.creatorId}</Link>
            {" · "}
            {formatDate(poll.createdAt)}
          </p>
        </div>
        <div className="actions">
          {isOwner ? (
            <Link className="button secondary" to={`/poll/${encodeURIComponent(poll.id)}/edit`}>
              Редактировать
            </Link>
          ) : null}
        </div>
      </section>

      <div className="split">
        <section className="card stack">
          {poll.imageUrl ? <img className="poll-image" src={poll.imageUrl} alt="" /> : null}
          <div className="tag-list">
            {(poll.tags || []).map((tag) => (
              <Link key={tag} className="tag" to={`/?tags=${encodeURIComponent(tag)}`}>
                {tagLabel(tag)}
              </Link>
            ))}
          </div>

          {isAuthenticated ? (
            <form className="stack" key={state.vote?.optionIds.join(",") || "no-vote"} onSubmit={submitVote}>
              <div className="option-list">
                {(poll.options || []).map((option) => (
                  <label className="vote-option" key={option.id}>
                    <input type={inputType} name="optionId" value={option.id} defaultChecked={selected.has(option.id)} disabled={voteBusy} />
                    <span>{option.text}</span>
                    <strong>{toCount(option.votesCount)}</strong>
                  </label>
                ))}
              </div>
              <div className="actions">
                <button type="submit" disabled={voteBusy}>
                  {voteBusy ? "Сохранение..." : state.vote?.hasVoted ? "Изменить голос" : "Проголосовать"}
                </button>
                <button className="secondary" type="button" disabled={!state.vote?.hasVoted || voteBusy} onClick={removeVote}>
                  Убрать голос
                </button>
              </div>
            </form>
          ) : (
            <div className="empty">Войдите, чтобы проголосовать. <Link to="/auth">Авторизация</Link></div>
          )}
        </section>

        <aside className="poll-side stack">
          <section className="card stack">
            <h3>Аналитика</h3>
            <PollAnalyticsCharts poll={poll} countries={state.countries} gender={state.gender} age={state.age} />
          </section>
        </aside>
      </div>
    </div>
  );
}

function settled<T>(result: PromiseSettledResult<T>): T | undefined {
  return result.status === "fulfilled" ? result.value : undefined;
}

function adjustStat<T extends { votes: number }>(
  items: T[],
  key: keyof T,
  value: string,
  delta: number,
  create: (value: string) => T,
): T[] {
  const normalized = value.trim();
  if (!normalized || !delta) return items;

  const next = items.map((item) => ({ ...item }));
  const index = next.findIndex((item) => {
    const current = String(item[key]);
    return key === "gender" ? normalizeGender(current) === normalized : current === normalized;
  });
  if (index >= 0) {
    next[index] = { ...next[index], votes: Math.max(0, next[index].votes + delta) };
    return next.filter((item) => item.votes > 0);
  }
  if (delta > 0) next.push(create(normalized));
  return next;
}
