import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
import type { FeedItem, Poll, PollOption } from "../types/domain";

export interface PollLiveState {
  totalVotes: number;
  options: { id: string; votesCount: number }[];
}

interface LiveUpdatesValue {
  setPollLive: (pollId: string, state: PollLiveState) => void;
  clearPollLive: (pollId: string) => void;
  reconcilePolls: (items: FeedItem[]) => void;
  mergeFeedItem: (item: FeedItem) => FeedItem;
  mergePoll: (poll: Poll) => Poll;
}

const LiveUpdatesContext = createContext<LiveUpdatesValue | null>(null);

function applyLiveState<T extends { options?: PollOption[]; totalVotes?: number }>(item: T, live?: PollLiveState): T {
  if (!live) return item;
  const votesById = new Map(live.options.map((option) => [option.id, option.votesCount]));
  return {
    ...item,
    totalVotes: live.totalVotes,
    options: (item.options || []).map((option) => ({
      ...option,
      votesCount: votesById.has(option.id) ? votesById.get(option.id)! : option.votesCount,
    })),
  };
}

export function pollToLive(poll: Pick<Poll, "options" | "totalVotes">): PollLiveState {
  return {
    totalVotes: Number(poll.totalVotes || 0),
    options: (poll.options || []).map((option) => ({
      id: option.id,
      votesCount: Number(option.votesCount || 0),
    })),
  };
}

export function LiveUpdatesProvider({ children }: { children: ReactNode }) {
  const [polls, setPolls] = useState<Record<string, PollLiveState>>({});

  const setPollLive = useCallback((pollId: string, state: PollLiveState) => {
    setPolls((current) => ({ ...current, [pollId]: state }));
  }, []);

  const clearPollLive = useCallback((pollId: string) => {
    setPolls((current) => {
      if (!current[pollId]) return current;
      const next = { ...current };
      delete next[pollId];
      return next;
    });
  }, []);

  const reconcilePolls = useCallback((items: FeedItem[]) => {
    setPolls((current) => {
      let changed = false;
      const next = { ...current };
      for (const item of items) {
        const live = next[item.id];
        if (!live) continue;
        const serverTotal = Number(item.totalVotes || 0);
        if (serverTotal >= live.totalVotes) {
          delete next[item.id];
          changed = true;
        }
      }
      return changed ? next : current;
    });
  }, []);

  const mergeFeedItem = useCallback((item: FeedItem) => applyLiveState(item, polls[item.id]), [polls]);
  const mergePoll = useCallback((poll: Poll) => applyLiveState(poll, polls[poll.id]), [polls]);

  const value = useMemo(
    () => ({ setPollLive, clearPollLive, reconcilePolls, mergeFeedItem, mergePoll }),
    [clearPollLive, mergeFeedItem, mergePoll, reconcilePolls, setPollLive],
  );

  return <LiveUpdatesContext.Provider value={value}>{children}</LiveUpdatesContext.Provider>;
}

export function useLiveUpdates() {
  const value = useContext(LiveUpdatesContext);
  if (!value) throw new Error("useLiveUpdates must be used inside LiveUpdatesProvider");
  return value;
}
