export type PollType = "POLL_TYPE_SINGLE_CHOICE" | "POLL_TYPE_MULTIPLE_CHOICE";

export interface AuthTokens {
  accessToken: string;
  refreshToken: string;
  expiresInSeconds: number;
}

export interface User {
  id: string;
  nickname: string;
  email: string;
  country: string;
  gender: string;
  birthYear: number;
  createdAt: string;
}

export interface PublicUserProfile {
  id: string;
  nickname: string;
  followersCount: number;
  followingCount: number;
  isFollowing: boolean;
}

export interface PollOption {
  id: string;
  text: string;
  votesCount: number;
}

export interface Poll {
  id: string;
  creatorId: string;
  question: string;
  type: PollType;
  createdAt: string;
  totalVotes: number;
  options: PollOption[];
  tags: string[];
  imageUrl?: string;
}

export interface FeedAuthor {
  id: string;
  nickname: string;
}

export interface FeedItem extends Poll {
  author?: FeedAuthor;
}

export interface PageMeta {
  nextCursor?: string;
  hasMore?: boolean;
  limit?: number;
}

export interface FeedResponse {
  items: FeedItem[];
  page?: PageMeta;
}

export interface Tag {
  id: string;
  name: string;
  createdAt: string;
}

export interface VoteState {
  pollId: string;
  hasVoted: boolean;
  optionIds: string[];
  votedAt?: string;
}

export interface OptionStat {
  optionId: string;
  votes: number;
}

export interface CountryStat {
  country: string;
  votes: number;
}

export interface GenderStat {
  gender: string;
  votes: number;
}

export interface AgeStat {
  ageRange: string;
  votes: number;
}

export interface PollAnalytics {
  pollId: string;
  totalVotes: number;
  options: OptionStat[];
  countries: CountryStat[];
  gender: GenderStat[];
  age: AgeStat[];
}
