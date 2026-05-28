export interface PredefinedTag {
  id: string;
  label: string;
}

export const PREDEFINED_TAGS: PredefinedTag[] = [
  { id: "entertainment", label: "Развлечения" },
  { id: "science", label: "Наука" },
  { id: "food", label: "Еда" },
  { id: "sport", label: "Спорт" },
  { id: "technology", label: "Технологии" },
  { id: "travel", label: "Путешествия" },
  { id: "music", label: "Музыка" },
  { id: "movies", label: "Кино" },
  { id: "health", label: "Здоровье" },
  { id: "society", label: "Общество" },
  { id: "education", label: "Образование" },
  { id: "business", label: "Бизнес" },
  { id: "games", label: "Игры" },
  { id: "fashion", label: "Мода" },
  { id: "nature", label: "Природа" },
];

const tagLabelMap = new Map(PREDEFINED_TAGS.map((tag) => [tag.id, tag.label]));

export function tagLabel(id: string): string {
  return tagLabelMap.get(id) || id;
}

export function isPredefinedTag(id: string): boolean {
  return tagLabelMap.has(id);
}
