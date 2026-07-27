import type { HuangliData } from '../types';
import { tianGanDiZhi, shichenTime } from '../types';

interface DayViewProps {
  huangliData: HuangliData | null;
  currentDay: number;
  currentMonth: number;
  weekDay: number;
  weekDays: string[];
  lunarYear: string;
  dateStr: string;
  zodiacSign: string;
  yearZhi: string;
  zodiacIcon: string;
  directions: { name: string; direction: string }[];
}

export function DayView({
  huangliData, currentDay, weekDay, weekDays,
  lunarYear, dateStr, zodiacSign, yearZhi, zodiacIcon, directions
}: DayViewProps) {
  if (!huangliData) {
    return (
      <div className="card text-center py-12">
        <p className="text-jade/60">暂无黄历数据</p>
      </div>
    );
  }

  return (
    <div className="space-y-4 animate-fade-in-up">
      {/* 大日期 */}
      <div className="consultation-sheet text-center py-6 corner-decor">
        <div className="font-serif text-7xl md:text-8xl text-crimson leading-none">{currentDay}</div>
        <div className="mt-2 text-jade text-sm">
          {lunarYear} · {dateStr} · 星期{weekDays[weekDay]}
        </div>
        <div className="flex items-center justify-center gap-4 mt-3">
          <span className="text-xs px-2.5 py-1 bg-warm-white/80 text-jade rounded-full">{zodiacSign}</span>
          <span className="text-2xl">{zodiacIcon}</span>
          <span className="text-xs px-2.5 py-1 bg-warm-white/80 text-jade rounded-full">{yearZhi}年</span>
        </div>
      </div>

      {/* 宜忌 */}
      <div className="grid grid-cols-2 gap-3">
        <div className="card">
          <div className="flex items-center gap-2 mb-3">
            <span className="w-7 h-7 bg-jade text-white rounded-full flex items-center justify-center text-sm font-bold">宜</span>
            <span className="text-jade text-sm font-medium">宜</span>
          </div>
          <div className="flex flex-wrap gap-1.5">
            {huangliData.yi?.length ? (
              huangliData.yi.map((item, i) => (
                <span key={i} className="text-xs px-2 py-0.5 bg-jade/8 text-jade rounded-full">{item}</span>
              ))
            ) : (
              <span className="text-ink-light/40 text-xs">诸事不宜</span>
            )}
          </div>
        </div>
        <div className="card">
          <div className="flex items-center gap-2 mb-3">
            <span className="w-7 h-7 bg-crimson text-white rounded-full flex items-center justify-center text-sm font-bold">忌</span>
            <span className="text-crimson text-sm font-medium">忌</span>
          </div>
          <div className="flex flex-wrap gap-1.5">
            {huangliData.ji?.length ? (
              huangliData.ji.map((item, i) => (
                <span key={i} className="text-xs px-2 py-0.5 bg-crimson/8 text-crimson rounded-full">{item}</span>
              ))
            ) : (
              <span className="text-ink-light/40 text-xs">诸事皆宜</span>
            )}
          </div>
        </div>
      </div>

      {/* 冲煞与值神 */}
      <div className="grid grid-cols-2 gap-3">
        <div className="card text-center">
          <p className="text-xs text-jade mb-1">冲煞</p>
          <p className="text-ink text-sm">{huangliData.chong || ''}{huangliData.sha || ''}</p>
        </div>
        <div className="card text-center">
          <p className="text-xs text-jade mb-1">值神</p>
          <p className="text-ink text-sm">{huangliData.zhiShen || ''}</p>
        </div>
      </div>

      {/* 吉神方位 */}
      <div className="card">
        <h3 className="section-title mb-3">吉神方位</h3>
        <div className="grid grid-cols-5 gap-2 text-center">
          {directions.map((dir, i) => (
            <div key={i} className="p-2 bg-warm-white/60 rounded-lg">
              <div className="text-xs text-jade">{dir.name}</div>
              <div className="text-sm text-ink font-medium mt-0.5">{dir.direction}</div>
            </div>
          ))}
        </div>
      </div>

      {/* 吉神凶神 */}
      <div className="grid grid-cols-2 gap-3">
        <div className="card">
          <h3 className="text-jade text-sm font-medium mb-2">吉神宜趋</h3>
          <p className="text-xs text-ink leading-relaxed">
            {huangliData.jiShen?.join(' ') || huangliData.jieShen || '—'}
          </p>
        </div>
        <div className="card">
          <h3 className="text-crimson text-sm font-medium mb-2">凶神宜忌</h3>
          <p className="text-xs text-ink leading-relaxed">
            {huangliData.xiongSha?.join(' ') || huangliData.zhangSong || '—'}
          </p>
        </div>
      </div>

      {/* 详细信息 */}
      <div className="grid grid-cols-3 gap-3">
        <div className="card text-center">
          <p className="text-xs text-jade mb-1">胎神占方</p>
          <p className="text-xs text-ink">{huangliData.taiShen || '—'}</p>
        </div>
        <div className="card text-center">
          <p className="text-xs text-jade mb-1">彭祖百忌</p>
          <p className="text-xs text-ink">{huangliData.pengZhu || '—'}</p>
        </div>
        <div className="card text-center">
          <p className="text-xs text-jade mb-1">五行日</p>
          <p className="text-xs text-ink">{huangliData.dayWuxing || '—'}</p>
        </div>
      </div>

      {/* 建除与星宿 */}
      <div className="grid grid-cols-2 gap-3">
        <div className="card text-center">
          <p className="text-xs text-jade mb-1">建除十二神</p>
          <p className="text-ink text-sm">{huangliData.jianChu || '—'}日 · {huangliData.zhiShen || ''}执位</p>
        </div>
        <div className="card text-center">
          <p className="text-xs text-jade mb-1">二十八星宿</p>
          <p className="text-ink text-sm">{huangliData.xingXiu || '—'}宿</p>
        </div>
      </div>

      {/* 时辰吉凶 */}
      <div className="card">
        <h3 className="section-title mb-3">时辰吉凶</h3>
        <div className="grid grid-cols-4 sm:grid-cols-6 gap-2">
          {(huangliData.hours?.length ? huangliData.hours : tianGanDiZhi.map((_, i) => ({ timeRange: shichenTime[i], jiXiong: '' }))).map((hour, i) => (
            <div key={i} className="text-center p-2 bg-warm-white/60 rounded-lg">
              <div className="text-xs font-medium text-ink">{tianGanDiZhi[i]}</div>
              <div className="text-[10px] text-jade/60 mt-0.5">{hour.timeRange}</div>
              <div className={`text-xs font-medium mt-0.5 ${hour.jiXiong === '吉' ? 'text-jade' : hour.jiXiong === '凶' ? 'text-crimson' : 'text-ink-light/40'}`}>
                {hour.jiXiong || '—'}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
