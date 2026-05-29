import type {
  AuthTokens,
  FeedResponse,
  Poll,
  PollAnalytics,
  PublicUserProfile,
  User,
  VoteState,
} from "../types/domain";
import { getViewerId } from "../utils/viewerId";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "";

export interface SessionSnapshot {
  accessToken: string;
  refreshToken: string;
}

export interface ApiClientOptions {
  getSession: () => SessionSnapshot;
  setTokens: (tokens: AuthTokens) => void;
  clearSession: () => void;
}

type ApiOptions = {
  method?: string;
  body?: unknown;
  auth?: boolean;
  skipRefresh?: boolean;
  headers?: Record<string, string>;
};

export class ApiClient {
  constructor(private readonly options: ApiClientOptions) {}

  register(body: Record<string, unknown>) {
    return this.request<{ tokens: AuthTokens }>("/v1/auth/register", { method: "POST", body });
  }

  login(body: Record<string, unknown>) {
    return this.request<{ tokens: AuthTokens }>("/v1/auth/login", { method: "POST", body });
  }

  logout(refreshToken: string) {
    return this.request("/v1/auth/logout", { method: "POST", auth: true, body: { refreshToken } });
  }

  me() {
    return this.request<{ user: User }>("/v1/users/me", { auth: true });
  }

  updateMe(body: Partial<User>) {
    return this.request<{ user: User }>("/v1/users/me", { method: "PATCH", auth: true, body });
  }

  profile(userId: string, auth = false) {
    return this.request<{ profile: PublicUserProfile }>(`/v1/profiles/${encodeURIComponent(userId)}`, { auth });
  }

  follow(userId: string, isFollowing: boolean) {
    return this.request(`/v1/users/${encodeURIComponent(userId)}:follow`, {
      method: isFollowing ? "DELETE" : "POST",
      auth: true,
      body: isFollowing ? undefined : {},
    });
  }

  feed(path: string, search: string, auth = false) {
    return this.request<FeedResponse>(`${path}?${search}`, { auth });
  }

  recordFeedImpressions(feedItemIds: string[]) {
    const viewerKey = getViewerId();
    if (!viewerKey || feedItemIds.length === 0) {
      return Promise.resolve({ recorded: 0 });
    }
    return this.request<{ recorded: number }>("/v1/feed/impressions", {
      method: "POST",
      body: { viewerKey, feedItemIds },
      headers: { "X-Viewer-Key": viewerKey },
    });
  }

  userPolls(userId: string, search: string) {
    return this.request<FeedResponse>(`/v1/feed/user/${encodeURIComponent(userId)}?${search}`);
  }

  poll(id: string) {
    return this.request<{ poll: Poll }>(`/v1/polls/${encodeURIComponent(id)}`);
  }

  createPoll(body: Record<string, unknown>) {
    return this.request<{ poll: Poll }>("/v1/polls", { method: "POST", auth: true, body });
  }

  updatePoll(id: string, body: Record<string, unknown>) {
    return this.request<{ poll: Poll }>(`/v1/polls/${encodeURIComponent(id)}`, { method: "PATCH", auth: true, body });
  }

  deletePoll(id: string) {
    return this.request(`/v1/polls/${encodeURIComponent(id)}`, { method: "DELETE", auth: true });
  }

  vote(pollId: string, optionIds: string[]) {
    return this.request(`/v1/polls/${encodeURIComponent(pollId)}/vote`, { method: "POST", auth: true, body: { optionIds } });
  }

  removeVote(pollId: string) {
    return this.request(`/v1/polls/${encodeURIComponent(pollId)}/vote`, { method: "DELETE", auth: true });
  }

  userVote(pollId: string) {
    return this.request<VoteState>(`/v1/polls/${encodeURIComponent(pollId)}/vote`, { auth: true });
  }

  analytics(pollId: string) {
    return this.request<PollAnalytics>(`/v1/polls/${encodeURIComponent(pollId)}/analytics`);
  }

  analyticsSection<T>(pollId: string, section: "countries" | "gender" | "age") {
    return this.request<{ items: T[] }>(`/v1/polls/${encodeURIComponent(pollId)}/analytics/${section}`);
  }

  uploadUrl(file: File) {
    return this.request<{ uploadUrl: string; imageUrl: string }>("/v1/polls/images:upload-url", {
      method: "POST",
      auth: true,
      body: { filename: file.name, contentType: file.type || "application/octet-stream", sizeBytes: file.size },
    });
  }

  async uploadImage(file: File): Promise<string> {
    const upload = await this.uploadUrl(file);
    const response = await fetch(`/__upload_proxy?uploadUrl=${encodeURIComponent(upload.uploadUrl)}`, {
      method: "POST",
      headers: { "X-Upload-Content-Type": file.type || "application/octet-stream" },
      body: file,
    });
    if (!response.ok) {
      const text = await response.text().catch(() => "");
      throw new Error(`Не удалось загрузить изображение: ${text || response.status}`);
    }
    return upload.imageUrl;
  }

  private async request<T = Record<string, never>>(path: string, options: ApiOptions = {}): Promise<T> {
    const { method = "GET", body, auth = false, skipRefresh = false } = options;
    const headers: Record<string, string> = { Accept: "application/json", ...options.headers };
    const session = this.options.getSession();
    if (body !== undefined) headers["Content-Type"] = "application/json";
    if (auth && session.accessToken) headers.Authorization = `Bearer ${session.accessToken}`;

    const response = await fetch(API_BASE_URL + path, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
    const payload = await parseResponse(response);

    if (response.status === 401 && auth && session.refreshToken && !skipRefresh) {
      try {
        const refreshed = await this.request<{ tokens: AuthTokens }>("/v1/auth/refresh", {
          method: "POST",
          body: { refreshToken: session.refreshToken },
          skipRefresh: true,
        });
        this.options.setTokens(refreshed.tokens);
        return this.request<T>(path, { ...options, skipRefresh: true });
      } catch {
        this.options.clearSession();
      }
    }

    if (!response.ok) throw new Error(readableError(payload, response.status));
    return payload as T;
  }
}

async function parseResponse(response: Response): Promise<unknown> {
  const text = await response.text();
  if (!text) return {};
  try {
    return JSON.parse(text);
  } catch {
    return { message: text };
  }
}

function readableError(payload: unknown, status: number): string {
  const object = payload && typeof payload === "object" ? (payload as Record<string, unknown>) : {};
  const raw = String(object.message || object.error || `Ошибка ${status}`);
  const clean = raw.replace(/^rpc error: code = \w+ desc = /, "");
  const messages: Record<string, string> = {
    "missing or invalid bearer token": "Войдите в аккаунт.",
    "invalid token": "Сессия истекла. Войдите снова.",
    "email already exists": "Email уже занят.",
    "nickname already exists": "Никнейм уже занят.",
    "cannot follow self": "Нельзя подписаться на себя.",
    "not found": "Запись не найдена.",
  };
  return messages[clean] || clean;
}
