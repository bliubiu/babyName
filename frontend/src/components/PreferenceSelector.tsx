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
    <div className="pt-6 border-t border-paper space-y-6">

      {/* ---- 经典来源 ---- */}
      <section>
        <h3 className="form-label mb-1">经典来源</h3>
        <p className="text-xs text-ink-light/50 mb-3">选择特定经典文本作为起名来源（可不选，默认综合所有经典）</p>
        <div className="flex flex-wrap gap-2">
          {classicOptions.map((option) => (
            <button
              key={option.id}
              type="button"
              onClick={() => onSourceClassicChange(sourceClassic === option.id ? '' : option.id)}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all duration-200 ${
                sourceClassic === option.id
                  ? 'bg-crimson text-white shadow-sm'
                  : 'bg-paper text-ink-light hover:bg-crimson/10 hover:text-crimson border border-paper-edge/30'
              }`}
            >
              {option.label}
            </button>
          ))}
        </div>
        {sourceClassic && (
          <p className="text-xs text-jade mt-2">
            当前已选择：<span className="font-semibold">{sourceClassic}</span>
            ，起名将优先从此来源选字
          </p>
        )}
      </section>

      {/* ---- 个性补充 ---- */}
      <section>
        <h3 className="form-label mb-3">个性补充</h3>
        <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-4">
          <label className="flex-shrink-0 text-ink-light text-sm">
            寓意关键词：
          </label>
          <input
            type="text"
            className="input-field flex-1 max-w-md"
            placeholder="输入关键词，如：清莲、竹韵、明德、鹤影"
            value={keywords}
            onChange={(e) => onKeywordsChange(e.target.value)}
          />
        </div>
      </section>

    </div>
  );
}
