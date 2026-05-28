export const FEED_MODES = [
  { path: "/", label: "Лента", apiPath: "/v1/feed", tags: true, auth: false },
  { path: "/trending", label: "Тренды", apiPath: "/v1/feed/trending", tags: false, auth: false },
  { path: "/following", label: "Подписки", apiPath: "/v1/feed/following", tags: false, auth: true },
] as const;

export type FeedMode = (typeof FEED_MODES)[number];

export function feedModeByPath(pathname: string): FeedMode {
  return FEED_MODES.find((item) => item.path === pathname) || FEED_MODES[0];
}
