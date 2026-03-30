'use client';

interface PreferenceSelectorProps {
  preferences: string[];
  onToggle: (preference: string) => void;
  keywords: string;
  onKeywordsChange: (keywords: string) => void;
}

const preferenceOptions = [
  { id: 'poetry', emoji: '📜', label: '诗词起名', color: 'text-red-500' },
  { id: 'sound', emoji: '🔊', label: '音形义起名', color: 'text-purple-500' },
  { id: 'zodiac', emoji: '🐲', label: '生肖起名', color: 'text-green-500' },
  { id: 'data', emoji: '📊', label: '大数据起名', color: 'text-blue-500' },
  { id: 'expectation', emoji: '✨', label: '期望起名', color: 'text-yellow-500' },
  { id: 'plant', emoji: '🌸', label: '花草起名', color: 'text-pink-500' },
];

export function PreferenceSelector({ preferences, onToggle, keywords, onKeywordsChange }: PreferenceSelectorProps) {
  return (
    <div className="pt-6 border-t border-stone-300">
      <h3 className="text-lg font-medium text-stone-700 mb-4">偏好选择</h3>

      <div className="flex flex-wrap gap-2 sm:gap-3 mb-4">
        {preferenceOptions.map((option) => (
          <button
            key={option.id}
            type="button"
            onClick={() => onToggle(option.id)}
            className={`preference-tag px-3 sm:px-4 py-2 rounded-full border-2 border-amber-200 bg-amber-50 text-stone-700 hover:border-amber-400 transition-colors flex items-center gap-2 text-sm sm:text-base ${
              preferences.includes(option.id) ? 'active' : ''
            }`}
          >
            <span className={option.color}>{option.emoji}</span> {option.label}
          </button>
        ))}
      </div>

      <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-4">
        <label className="flex-shrink-0 text-stone-600 text-sm flex items-center gap-1">
          <span className="text-yellow-500">🌙</span> 个性补充 (诗词/花草关键词):
        </label>
        <input
          type="text"
          className="flex-1 px-4 py-2 rounded-full border border-stone-300 bg-white focus:outline-none focus:border-amber-500 text-sm"
          placeholder="如: 清莲, 明德, 鹤影, 竹韵, 静秋"
          value={keywords}
          onChange={(e) => onKeywordsChange(e.target.value)}
        />
      </div>
    </div>
  );
}
