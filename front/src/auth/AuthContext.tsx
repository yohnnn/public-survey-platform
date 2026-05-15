import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
import { ApiClient, type SessionSnapshot } from "../api/client";
import type { AuthTokens, User } from "../types/domain";

const STORAGE_KEY = "psp-user-front";

interface StoredSession extends SessionSnapshot {
  me: User | null;
}

interface AuthContextValue {
  accessToken: string;
  refreshToken: string;
  me: User | null;
  api: ApiClient;
  isAuthenticated: boolean;
  setTokens: (tokens: AuthTokens) => void;
  setMe: (user: User | null) => void;
  clearSession: () => void;
  loadMe: (force?: boolean) => Promise<User | null>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<StoredSession>(() => readSession());

  const persist = useCallback((next: StoredSession) => {
    setSession(next);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
  }, []);

  const setTokens = useCallback(
    (tokens: AuthTokens) => {
      setSession((current) => {
        const next = {
          ...current,
          accessToken: tokens.accessToken || "",
          refreshToken: tokens.refreshToken || current.refreshToken || "",
        };
        localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
        return next;
      });
    },
    [],
  );

  const clearSession = useCallback(() => {
    persist({ accessToken: "", refreshToken: "", me: null });
  }, [persist]);

  const api = useMemo(
    () =>
      new ApiClient({
        getSession: () => readSession(),
        setTokens,
        clearSession,
      }),
    [clearSession, setTokens],
  );

  const setMe = useCallback(
    (user: User | null) => {
      setSession((current) => {
        const next = { ...current, me: user };
        localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
        return next;
      });
    },
    [],
  );

  const loadMe = useCallback(
    async (force = false) => {
      if (!readSession().accessToken) return null;
      if (session.me && !force) return session.me;
      try {
        const response = await api.me();
        setMe(response.user);
        return response.user;
      } catch (error) {
        clearSession();
        if (force) throw error;
        return null;
      }
    },
    [api, clearSession, session.me, setMe],
  );

  const logout = useCallback(async () => {
    const refreshToken = readSession().refreshToken;
    try {
      if (refreshToken) await api.logout(refreshToken);
    } finally {
      clearSession();
    }
  }, [api, clearSession]);

  const value = useMemo<AuthContextValue>(
    () => ({
      ...session,
      api,
      isAuthenticated: Boolean(session.accessToken),
      setTokens,
      setMe,
      clearSession,
      loadMe,
      logout,
    }),
    [api, clearSession, loadMe, logout, session, setMe, setTokens],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used inside AuthProvider");
  return value;
}

function readSession(): StoredSession {
  const fallback: StoredSession = { accessToken: "", refreshToken: "", me: null };
  const raw = localStorage.getItem(STORAGE_KEY);
  if (!raw) return fallback;
  try {
    const parsed = JSON.parse(raw) as Partial<StoredSession>;
    return {
      accessToken: parsed.accessToken || "",
      refreshToken: parsed.refreshToken || "",
      me: parsed.me || null,
    };
  } catch {
    return fallback;
  }
}
