const STORAGE_KEY = "psp_viewer_id";

export function getViewerId(): string {
  if (typeof window === "undefined") return "";
  const existing = window.localStorage.getItem(STORAGE_KEY);
  if (existing) return existing;

  const id = typeof crypto !== "undefined" && "randomUUID" in crypto ? crypto.randomUUID() : `anon-${Date.now()}`;
  window.localStorage.setItem(STORAGE_KEY, id);
  return id;
}
