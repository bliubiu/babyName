'use client';

interface PreferenceSelectorProps {
  keywords: string;
  onKeywordsChange: (keywords: string) => void;
  sourceClassic: string;
  onSourceClassicChange: (source: string) => void;
}

const classicOptions = [
  { id: '诗经', label: '诗经' },
  { id: '楚辞', label: '楚辞' },
  { id: '唐诗宋词', label: '唐诗宋词' },
  { id: '古文观止', label: '古文观止' },
  { id: '论语', label: '论语' },
  { id: '孟子', label: '孟子' },
  { id: '三字经', label: '三字经' },
  { id: '千字文', label: '千字文' },
  { id: '声律启蒙', label: '声律启蒙' },
];

export function PreferenceSelector({
  keywords, onKeywordsChange,
  sourceClassic, onSourceClassicChange,
}: PreferenceSelectorProps) {
  return (
    <div className="home-preferences">
      {/* ---- 经典来源 ---- */}
      <section className="home-subsection">
        <h3 className="home-subsection-title">经典来源</h3>
        <p className="home-helper mb-3">选择特定经典文本作为起名来源（可不选，默认综合所有经典）</p>
        <div className="flex flex-wrap gap-2">
          {classicOptions.map((option) => (
            <button
              key={option.id}
              type="button"
              onClick={() => onSourceClassicChange(sourceClassic === option.id ? '' : option.id)}
              aria-pressed={sourceClassic === option.id}
              className={`home-chip ${sourceClassic === option.id ? 'home-chip-active' : ''}`}
            >
              {option.label}
            </button>
          ))}
        </div>
        {sourceClassic && (
          <p className="home-chip-hint">
            当前已选择：<span className="font-semibold">{sourceClassic}</span>
            ，起名将优先从此来源选字
          </p>
        )}
      </section>

      {/* ---- 个性补充 ---- */}
      <section className="home-subsection">
        <h3 className="home-subsection-title">个性补充</h3>
        <label htmlFor="keywords-input" className="home-helper block mb-2">寓意关键词</label>
        <input
          id="keywords-input"
          type="text"
          className="input-field"
          placeholder="输入关键词，如：清莲、竹韵、明德、鹤影"
          value={keywords}
          onChange={(e) => onKeywordsChange(e.target.value)}
        />
      </section>
    </div>
  );
}
