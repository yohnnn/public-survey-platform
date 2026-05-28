import { SlidersHorizontal, X } from "lucide-react";
import { PREDEFINED_TAGS, tagLabel } from "../data/tags";

interface FeedFiltersProps {
  activeTags: string[];
  open: boolean;
  onToggle: () => void;
  onSelect: (tagId: string) => void;
  onRemove: (tagId: string) => void;
  onClear: () => void;
}

export function FeedFilters({ activeTags, open, onToggle, onSelect, onRemove, onClear }: FeedFiltersProps) {
  const hasFilter = activeTags.length > 0;

  return (
    <div className="feed-filters">
      <div className="feed-filters-bar">
        <button type="button" className={`filter-trigger${open ? " open" : ""}${hasFilter ? " active" : ""}`} onClick={onToggle}>
          <SlidersHorizontal size={17} />
          <span>Фильтры</span>
          {hasFilter ? <span className="filter-badge">{activeTags.length}</span> : null}
        </button>

        {hasFilter ? (
          <div className="active-filter-chips">
            {activeTags.map((tagId) => (
              <div className="active-filter-chip" key={tagId}>
                <span>{tagLabel(tagId)}</span>
                <button type="button" className="filter-chip-clear" aria-label={`Убрать ${tagLabel(tagId)}`} onClick={() => onRemove(tagId)}>
                  <X size={14} />
                </button>
              </div>
            ))}
            <button type="button" className="ghost compact" onClick={onClear}>
              Сбросить все
            </button>
          </div>
        ) : null}
      </div>

      <div className={`filter-panel${open ? " open" : ""}`}>
        <div className="filter-panel-head">
          <div>
            <strong>Темы</strong>
            <p className="muted small">Покажем опросы с любой из выбранных тем</p>
          </div>
          <button type="button" className="ghost compact" onClick={onToggle}>
            Закрыть
          </button>
        </div>
        <div className="filter-chip-grid">
          <button
            type="button"
            className={`filter-chip${!hasFilter ? " selected" : ""}`}
            onClick={onClear}
          >
            Все темы
          </button>
          {PREDEFINED_TAGS.map((tag) => (
            <button
              key={tag.id}
              type="button"
              className={`filter-chip${activeTags.includes(tag.id) ? " selected" : ""}`}
              onClick={() => onSelect(tag.id)}
            >
              {tag.label}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}
