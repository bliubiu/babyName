'use client';

import React, { useState, useMemo, useCallback } from 'react';
import { getHuangli, getLunarCalendar } from '@/lib/api';
import { useQueries } from '@tanstack/react-query';
import { Spinner } from '@/components/Spinner';
import Navigation from '@/components/Navigation';
import type { HuangliData, LunarCalendarData } from './types';
import { zodiacIcons, getZodiacSign } from './types';
import { DayView } from './components/DayView';
import { WannianliView } from './components/WannianliView';

const HuangliPage = () => {
  const [selectedDate, setSelectedDate] = useState<Date>(new Date());
  const [activeTab, setActiveTab] = useState<'day' | 'month'>('day');

  const year = selectedDate.getFullYear();
  const month = selectedDate.getMonth() + 1;
  const day = selectedDate.getDate();

  const [huangliQuery, lunarQuery] = useQueries({
    queries: [
      { queryKey: ['huangli', year, month, day], queryFn: () => getHuangli(year, month, day) },
      { queryKey: ['lunar', year, month, day], queryFn: () => getLunarCalendar(year, month, day) },
    ],
  });

  const huangliData: HuangliData | null = huangliQuery.data?.data ?? null;
  const lunarData = (lunarQuery.data?.data ?? null) as LunarCalendarData | null;
  const loading = huangliQuery.isLoading || lunarQuery.isLoading;
  const error = huangliQuery.error || lunarQuery.error ? '获取数据失败，请重试' : null;

  const goToPrevDay = useCallback(() => {
    const d = new Date(selectedDate);
    d.setDate(d.getDate() - 1);
    setSelectedDate(d);
  }, [selectedDate]);

  const goToNextDay = useCallback(() => {
    const d = new Date(selectedDate);
    d.setDate(d.getDate() + 1);
    setSelectedDate(d);
  }, [selectedDate]);

  const goToToday = useCallback(() => setSelectedDate(new Date()), []);

  const dateInfo = useMemo(() => {
    const y = selectedDate.getFullYear();
    const m = selectedDate.getMonth() + 1;
    const d = selectedDate.getDate();
    const w = selectedDate.getDay();
    const weekDays = ['日', '一', '二', '三', '四', '五', '六'];
    return { year: y, month: m, day: d, weekDay: w, weekDays };
  }, [selectedDate]);

  const { year: currentYear, month: currentMonth, day: currentDay, weekDay, weekDays } = dateInfo;

  const lunarInfo = useMemo(() => {
    const zodiacSign = getZodiacSign(currentMonth, currentDay);
    const yearZhi = lunarData?.yearZhi || '';
    const zodiacIcon = zodiacIcons[yearZhi] || '🐴';
    return {
      lunarYear: lunarData?.lunarYearName || '',
      lunarMonth: lunarData?.lunarMonthName || '',
      lunarDate: lunarData?.lunarDayName || '',
      dateStr: lunarData?.dateStr || '',
      zodiacSign, yearZhi, zodiacIcon
    };
  }, [currentMonth, currentDay, lunarData]);

  const { lunarYear, dateStr, zodiacSign, yearZhi, zodiacIcon } = lunarInfo;

  const directions = useMemo(() => [
    { name: '喜神', direction: huangliData?.xiShen || '' },
    { name: '福神', direction: huangliData?.fuShen || '' },
    { name: '财神', direction: huangliData?.caiShen || '' },
    { name: '阳贵', direction: huangliData?.yangGui || '' },
    { name: '阴贵', direction: huangliData?.yinGui || '' },
  ], [huangliData]);

  return (
    <main className="min-h-screen py-6 md:py-8 px-4 md:px-6 cloud-bg">
      <div className="max-w-4xl mx-auto">
        <Navigation showBackButton={true} showHistory={true} showFavorites={true} />

        {/* 标题 */}
        <div className="text-center mb-6 md:mb-8 animate-fade-in-up">
          <h1 className="font-serif text-2xl md:text-3xl text-ink ink-calligraphy">今日黄历</h1>
          <div className="flex items-center justify-center gap-4 mt-3">
            <div className="brush-divider max-w-[60px] flex-1" />
            <span className="text-paper-edge/40 text-xs tracking-[0.5em] font-serif">择吉</span>
            <div className="brush-divider max-w-[60px] flex-1" />
          </div>
        </div>

        {/* 日期选择器 */}
        <div className="card mb-4 animate-fade-in-up">
          <div className="flex items-center justify-between gap-3">
            <button onClick={goToPrevDay} className="p-2 rounded-lg hover:bg-warm-white/80 text-jade hover:text-crimson transition-all">
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
              </svg>
            </button>

            <div className="flex items-center gap-2">
              <select
                value={currentYear}
                onChange={(e) => setSelectedDate(new Date(parseInt(e.target.value), currentMonth - 1, currentDay))}
                className="bg-warm-white/60 text-ink border-none rounded-lg px-3 py-1.5 text-sm focus:outline-none cursor-pointer"
              >
                {Array.from({ length: 21 }).map((_, i) => (
                  <option key={2020 + i} value={2020 + i}>{2020 + i}年</option>
                ))}
              </select>
              <select
                value={currentMonth}
                onChange={(e) => setSelectedDate(new Date(currentYear, parseInt(e.target.value) - 1, currentDay))}
                className="bg-warm-white/60 text-ink border-none rounded-lg px-3 py-1.5 text-sm focus:outline-none cursor-pointer"
              >
                {Array.from({ length: 12 }).map((_, i) => (
                  <option key={i + 1} value={i + 1}>{i + 1}月</option>
                ))}
              </select>
            </div>

            <button onClick={goToToday} className="px-3 py-1.5 bg-crimson/10 text-crimson rounded-lg text-sm hover:bg-crimson/20 transition-all">
              今日
            </button>

            <button onClick={goToNextDay} className="p-2 rounded-lg hover:bg-warm-white/80 text-jade hover:text-crimson transition-all">
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
              </svg>
            </button>
          </div>
        </div>

        {/* 标签切换 */}
        <div className="flex mb-4 animate-fade-in-up" style={{ animationDelay: '0.1s' }}>
          <button
            onClick={() => setActiveTab('day')}
            className={`flex-1 py-2.5 text-center text-sm font-medium transition-all duration-200 rounded-l-xl ${
              activeTab === 'day'
                ? 'bg-crimson text-white'
                : 'bg-warm-white/60 text-jade hover:bg-warm-white'
            }`}
          >
            黄历信息
          </button>
          <button
            onClick={() => setActiveTab('month')}
            className={`flex-1 py-2.5 text-center text-sm font-medium transition-all duration-200 rounded-r-xl ${
              activeTab === 'month'
                ? 'bg-crimson text-white'
                : 'bg-warm-white/60 text-jade hover:bg-warm-white'
            }`}
          >
            万年历
          </button>
        </div>

        {loading ? (
          <div className="flex justify-center items-center py-20">
            <Spinner size="large" />
          </div>
        ) : error ? (
          <div className="card text-center py-8">
            <p className="text-crimson mb-3">{error}</p>
            <button
              onClick={() => { huangliQuery.refetch(); lunarQuery.refetch(); }}
              className="btn-primary"
            >
              重试
            </button>
          </div>
        ) : (
          activeTab === 'day' ? (
            <DayView
              huangliData={huangliData}
              currentDay={currentDay}
              currentMonth={currentMonth}
              weekDay={weekDay}
              weekDays={weekDays}
              lunarYear={lunarYear}
              dateStr={dateStr}
              zodiacSign={zodiacSign}
              yearZhi={yearZhi}
              zodiacIcon={zodiacIcon}
              directions={directions}
            />
          ) : (
            <WannianliView
              currentYear={currentYear}
              currentMonth={currentMonth}
              selectedDate={selectedDate}
              onDateSelect={setSelectedDate}
              onTodayClick={goToToday}
            />
          )
        )}
      </div>
    </main>
  );
};

export default HuangliPage;
