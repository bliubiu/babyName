import { useMemo } from 'react';
import type { LunarMonthCalendar } from '@/lib/api';

interface WannianliViewProps {
  currentYear: number;
  currentMonth: number;
  selectedDate: Date;
  monthLunar?: LunarMonthCalendar | null;
  onDateSelect: (date: Date) => void;
  onTodayClick: () => void;
}

export function WannianliView({ currentYear, currentMonth, selectedDate, monthLunar, onDateSelect, onTodayClick }: WannianliViewProps) {
  const daysInMonth = Array.from({ length: new Date(currentYear, currentMonth, 0).getDate() }, (_, i) => i + 1);
  const firstDayOfMonth = new Date(currentYear, currentMonth - 1, 1).getDay();
  const emptySlots = firstDayOfMonth === 0 ? 6 : firstDayOfMonth - 1;
  const weekDays = ['一', '二', '三', '四', '五', '六', '日'];

  // 公历法定节假日（与农历节日、节气并列展示）
  const getHoliday = (month: number, day: number): string => {
    const holidays: Record<string, string> = {
      '1-1': '元旦', '3-8': '妇女节', '5-1': '劳动节', '6-1': '儿童节',
      '8-1': '建军节', '9-10': '教师节', '10-1': '国庆节'
    };
    return holidays[`${month}-${day}`] || '';
  };

  // 由后端返回的逐日节气 / 农历节日标注构建 day -> 标注 的映射
  const lunarMarkMap = useMemo(() => {
    const map: Record<number, { solarTerm: string; lunarFestival: string }> = {};
    (monthLunar?.days ?? []).forEach((d) => {
      map[d.day] = { solarTerm: d.solarTerm, lunarFestival: d.lunarFestival };
    });
    return map;
  }, [monthLunar]);

  return (
    <div className="card animate-fade-in-up">
      {/* 月份导航 */}
      <div className="flex items-center justify-between mb-4">
        <button
          onClick={() => onDateSelect(new Date(currentYear, currentMonth - 2, 1))}
          className="p-2 rounded-lg hover:bg-warm-white/80 text-jade hover:text-crimson transition-all"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
        </button>
        <span className="font-serif text-lg text-ink">{currentYear}年{currentMonth}月</span>
        <button
          onClick={() => onDateSelect(new Date(currentYear, currentMonth, 1))}
          className="p-2 rounded-lg hover:bg-warm-white/80 text-jade hover:text-crimson transition-all"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
        </button>
      </div>

      {/* 星期标题 */}
      <div className="grid grid-cols-7 gap-1 mb-2">
        {weekDays.map((d, i) => (
          <div key={i} className={`text-center py-1.5 text-xs font-medium ${i >= 5 ? 'text-crimson/60' : 'text-jade/60'}`}>
            {d}
          </div>
        ))}
      </div>

      {/* 日历网格 */}
      <div className="grid grid-cols-7 gap-1">
        {Array.from({ length: emptySlots }).map((_, i) => (
          <div key={`empty-${i}`} className="h-14" />
        ))}
        {daysInMonth.map((day) => {
          const date = new Date(currentYear, currentMonth - 1, day);
          const isToday = date.toDateString() === new Date().toDateString();
          const isSelected = date.toDateString() === selectedDate.toDateString();
          const holiday = getHoliday(currentMonth, day);
          // 节气 / 农历节日：传统节日（重阳节等）优先，否则展示 24 节气（清明等）
          const mark = lunarMarkMap[day];
          const festivalMark = mark?.lunarFestival || mark?.solarTerm || '';
          const showMark = festivalMark || holiday;

          return (
            <div
              key={day}
              className={`h-14 flex flex-col items-center justify-center rounded-xl cursor-pointer transition-all ${
                isSelected
                  ? 'bg-crimson text-white'
                  : isToday
                    ? 'bg-crimson/10 text-crimson border border-crimson/20'
                    : 'hover:bg-warm-white/80 text-ink'
              }`}
              onClick={() => onDateSelect(date)}
              title={showMark ? showMark : undefined}
            >
              <div className="text-sm font-medium leading-none">{day}</div>
              {showMark && (
                <div className={`mt-0.5 text-[10px] leading-none truncate max-w-[3.5rem] text-center ${
                  isSelected
                    ? 'text-white/85'
                    : mark?.lunarFestival
                      ? 'text-amber-600'
                      : holiday
                        ? 'text-crimson/70'
                        : 'text-emerald-600'
                }`}>
                  {festivalMark || holiday}
                </div>
              )}
            </div>
          );
        })}
      </div>

      <div className="mt-4 text-center">
        <button onClick={onTodayClick} className="text-sm text-crimson hover:text-crimson-light transition-colors">
          回到今日
        </button>
      </div>
    </div>
  );
}
