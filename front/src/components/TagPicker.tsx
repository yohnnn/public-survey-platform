import { PREDEFINED_TAGS } from "../data/tags";

interface TagPickerProps {
  value: string[];
  onChange: (tags: string[]) => void;
  maxTags?: number;
  label?: string;
}

export function TagPicker({ value, onChange, maxTags = 3, label = "Теги" }: TagPickerProps) {
  function toggle(tagId: string) {
    if (value.includes(tagId)) {
      onChange(value.filter((id) => id !== tagId));
      return;
    }
    if (value.length >= maxTags) return;
    onChange([...value, tagId]);
  }

  return (
    <div className="tag-picker">
      <span className="tag-picker-label">{label}</span>
      <div className="tag-list">
        {PREDEFINED_TAGS.map((tag) => {
          const selected = value.includes(tag.id);
          const disabled = !selected && value.length >= maxTags;
          return (
            <button
              key={tag.id}
              type="button"
              className={`tag-pick${selected ? " selected" : ""}`}
              disabled={disabled}
              onClick={() => toggle(tag.id)}
            >
              {tag.label}
            </button>
          );
        })}
      </div>
      <p className="hint small">
        {value.length ? `Выбрано: ${value.length} из ${maxTags}` : `Можно выбрать до ${maxTags} тегов`}
      </p>
    </div>
  );
}
