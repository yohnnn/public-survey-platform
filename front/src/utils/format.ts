export function formatDate(value?: string): string {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleDateString("ru-RU", { day: "2-digit", month: "short", year: "numeric" });
}

export function toCount(value: unknown): string {
  const number = Math.abs(Number(value || 0));
  return Number.isNaN(number) ? String(value || 0) : number.toLocaleString("ru-RU");
}

export function initials(value?: string): string {
  return String(value || "?").trim().slice(0, 2).toUpperCase();
}

export function splitCSV(value?: string): string[] {
  return String(value || "")
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

export function detectAgeRange(birthYear: number, at = new Date()): string {
  if (!birthYear || birthYear <= 0) return "";
  const age = at.getFullYear() - birthYear;
  if (age <= 0) return "";
  if (age <= 24) return "18-24";
  if (age <= 34) return "25-34";
  if (age <= 44) return "35-44";
  return "45+";
}

export function buildListSearch(params: { cursor?: string; limit?: string; tags?: string | string[]; includeTags?: boolean }): string {
  const search = new URLSearchParams();
  search.set("limit", params.limit || "20");
  if (params.cursor) search.set("cursor", params.cursor);
  if (params.includeTags && params.tags) {
    const tagList = Array.isArray(params.tags) ? params.tags : splitCSV(params.tags);
    tagList.forEach((tag) => search.append("tags", tag));
  }
  return search.toString();
}
