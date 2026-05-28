import { X } from "lucide-react";

interface PollOptionsEditorProps {
  value: string[];
  onChange: (options: string[]) => void;
}

export function PollOptionsEditor({ value, onChange }: PollOptionsEditorProps) {
  function updateOption(index: number, text: string) {
    const next = [...value];
    next[index] = text;
    onChange(next);
  }

  function removeOption(index: number) {
    if (value.length <= 2) return;
    onChange(value.filter((_, i) => i !== index));
  }

  function addOption() {
    onChange([...value, ""]);
  }

  return (
    <div className="option-editor">
      <span className="option-editor-label">Варианты ответа</span>
      <div className="option-editor-list">
        {value.map((option, index) => (
          <div className="option-editor-row" key={index}>
            <input
              type="text"
              value={option}
              placeholder={`Вариант ${index + 1}`}
              maxLength={200}
              required={index < 2}
              onChange={(event) => updateOption(index, event.target.value)}
            />
            <button
              type="button"
              className="icon-button secondary"
              aria-label="Удалить вариант"
              disabled={value.length <= 2}
              onClick={() => removeOption(index)}
            >
              <X size={18} />
            </button>
          </div>
        ))}
      </div>
      <button type="button" className="secondary" onClick={addOption}>
        + Добавить вариант
      </button>
    </div>
  );
}
