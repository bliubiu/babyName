'use client';

import React, { useState, useMemo, useCallback } from 'react';
import { getHuangli, getLunarCalendar } from '@/lib/api';
import { useQueries } from '@tanstack/react-query';
import { Spinner } from '@/components/Spinner';
import Navigation from '@/components/Navigation';
import { getLunar } from 'chinese-lunar-calendar';

interface HuangliHour {
  timeRange: string;
  jiXiong: string;
}

interface HuangliData {
  chong?: string;
  sha?: string;
  liuYao?: string;
  zhiShen?: string;
  yi?: string[];
  ji?: string[];
  jiShen?: string[];
  jieShen?: string;
  hours?: HuangliHour[];
  xingXiu?: string;
  jianChu?: string;
  xiShen?: string;
  fuShen?: string;
  caiShen?: string;
  yangGui?: string;
  yinGui?: string;
  xiongSha?: string[];
  zhangSong?: string;
  taiShen?: string;
  pengZhu?: string;
  dayWuxing?: string;
}

interface Direction {
  name: string;
  direction: string;
  color: string;
}

// 生肖图标映射
const zodiacIcons: Record<string, string> = {
  '鼠': '🐭', '牛': '🐮', '虎': '🐯', '兔': '🐰',
  '龙': '🐲', '蛇': '🐍', '马': '🐴', '羊': '🐑',
  '猴': '🐵', '鸡': '🐔', '狗': '🐶', '猪': '🐷'
};

// 星座映射
const getZodiacSign = (month: number, day: number): string => {
  const dates = [20, 19, 21, 20, 21, 22, 23, 23, 23, 24, 23, 22];
  const signs = ['水瓶座', '双鱼座', '白羊座', '金牛座', '双子座', '巨蟹座',
                 '狮子座', '处女座', '天秤座', '天蝎座', '射手座', '摩羯座'];
  return day < dates[month - 1] ? signs[month - 1] : signs[month % 12];
};

// 十二天干地址
const tianGanDiZhi = ['甲子', '乙丑', '丙寅', '丁卯', '戊辰', '己巳', '庚午', '辛未', '壬申', '癸酉', '甲戌', '乙亥'];

// 十二时辰时间范围
const shichenTime = ['23:00-01:00', '01:00-03:00', '03:00-05:00', '05:00-07:00', '07:00-09:00', '09:00-11:00',
                     '11:00-13:00', '13:00-15:00', '15:00-17:00', '17:00-19:00', '19:00-21:00', '21:00-23:00'];

const HuangliPage = () => {
  // 状态管理
  const [selectedDate, setSelectedDate] = useState<Date>(new Date());
  const [activeTab, setActiveTab] = useState<'day' | 'month'>('day');

  // 使用 React Query 并发获取黄历和农历数据
  const year = selectedDate.getFullYear();
  const month = selectedDate.getMonth() + 1;
  const day = selectedDate.getDate();

  const [huangliQuery, lunarQuery] = useQueries({
    queries: [
      {
        queryKey: ['huangli', year, month, day],
        queryFn: () => getHuangli(year, month, day),
      },
      {
        queryKey: ['lunar', year, month, day],
        queryFn: () => getLunarCalendar(year, month, day),
      },
    ],
  });

  const huangliData: HuangliData | null = huangliQuery.data?.data ?? null;
  const lunarData = lunarQuery.data?.data ?? null;
  const loading = huangliQuery.isLoading || lunarQuery.isLoading;
  const error = huangliQuery.error || lunarQuery.error ? '获取数据失败，请重试' : null;

  // 日期导航 - 使用 useCallback 缓存
  const goToPrevDay = useCallback(() => {
    const newDate = new Date(selectedDate);
    newDate.setDate(newDate.getDate() - 1);
    setSelectedDate(newDate);
  }, [selectedDate]);

  const goToNextDay = useCallback(() => {
    const newDate = new Date(selectedDate);
    newDate.setDate(newDate.getDate() + 1);
    setSelectedDate(newDate);
  }, [selectedDate]);

  const goToToday = useCallback(() => {
    setSelectedDate(new Date());
  }, []);

  // 日期相关变量 - 使用 useMemo 缓存
  const dateInfo = useMemo(() => {
    const year = selectedDate.getFullYear();
    const month = selectedDate.getMonth() + 1;
    const day = selectedDate.getDate();
    const weekDay = selectedDate.getDay();
    const weekDays = ['日', '一', '二', '三', '四', '五', '六'];
    return { year, month, day, weekDay, weekDays };
  }, [selectedDate]);

  const { year: currentYear, month: currentMonth, day: currentDay, weekDay, weekDays } = dateInfo;

  // 农历信息 - 使用 useMemo 缓存
  const lunarInfo = useMemo(() => {
    const lunar = getLunar(currentYear, currentMonth, currentDay);
    const zodiacSign = getZodiacSign(currentMonth, currentDay);
    const yearZhi = lunarData?.yearZhi || '';
    const zodiacIcon = zodiacIcons[yearZhi] || '🐴';
    return {
      lunarYear: lunar.lunarYear,
      lunarMonth: lunar.lunarMonth,
      lunarDate: lunar.lunarDate,
      dateStr: lunar.dateStr,
      zodiacSign,
      yearZhi,
      zodiacIcon
    };
  }, [currentYear, currentMonth, currentDay, lunarData]);

  const { lunarYear, lunarMonth, lunarDate, dateStr, zodiacSign, zodiacIcon } = lunarInfo;

  // 后端数据 - 使用 useMemo 缓存
  const backendData = useMemo(() => {
    const todayXingxiu = huangliData?.xingXiu || '';
    const todayJianChu = huangliData?.jianChu || '';
    return { todayXingxiu, todayJianChu };
  }, [huangliData]);

  const { todayXingxiu, todayJianChu } = backendData;

  // 吉神方位数据 - 使用 useMemo 缓存
  const directions: Direction[] = useMemo(() => [
    { name: '喜神', direction: huangliData?.xiShen || '', color: 'text-green-600' },
    { name: '福神', direction: huangliData?.fuShen || '', color: 'text-green-600' },
    { name: '财神', direction: huangliData?.caiShen || '', color: 'text-green-600' },
    { name: '阳贵', direction: huangliData?.yangGui || '', color: 'text-green-600' },
    { name: '阴贵', direction: huangliData?.yinGui || '', color: 'text-green-600' },
  ], [huangliData]);

  // 获取时辰吉凶 - 使用 useCallback 缓存
  const getHourLuck = useCallback((index: number) => {
    if (huangliData?.hours && huangliData.hours[index]) {
      return huangliData.hours[index].jiXiong;
    }
    return '';
  }, [huangliData]);

  return (
    <div className="min-h-screen bg-gradient-to-b from-green-50 to-white">
      <Navigation showBackButton={true} showHistory={true} showFavorites={true} />
      
      <div className="container mx-auto px-4 py-4 max-w-6xl">
        {/* 顶部导航栏 */}
        <div className="bg-green-600 text-white rounded-t-lg p-4 flex flex-col md:flex-row items-center justify-between gap-4">
          <div className="flex items-center gap-2 w-full md:w-auto">
            <select 
              value={currentYear}
              onChange={(e) => {
                const newYear = parseInt(e.target.value);
                setSelectedDate(new Date(newYear, currentMonth - 1, currentDay));
              }}
              className="bg-green-700 text-white border-none rounded px-3 py-2 text-lg font-medium focus:outline-none cursor-pointer transition-all hover:bg-green-800"
            >
              {Array.from({ length: 21 }).map((_, i) => (
                <option key={2020 + i} value={2020 + i} className="text-gray-800">{2020 + i}年</option>
              ))}
            </select>
            <select
              value={currentMonth}
              onChange={(e) => {
                const newMonth = parseInt(e.target.value);
                setSelectedDate(new Date(currentYear, newMonth - 1, currentDay));
              }}
              className="bg-green-700 text-white border-none rounded px-3 py-2 text-lg font-medium focus:outline-none cursor-pointer transition-all hover:bg-green-800"
            >
              {Array.from({ length: 12 }).map((_, i) => (
                <option key={i + 1} value={i + 1} className="text-gray-800">{i + 1}月</option>
              ))}
            </select>
          </div>
          <button 
            onClick={goToToday}
            className="bg-white text-green-600 px-5 py-2 rounded-full text-sm font-medium hover:bg-green-50 transition-all transform hover:scale-105"
          >
            返回今日
          </button>
        </div>

        {/* 标签切换 */}
        <div className="bg-white border-b flex">
          <button 
            onClick={() => setActiveTab('day')}
            className={`flex-1 py-4 text-center font-medium transition-all duration-300 ${
              activeTab === 'day' 
                ? 'text-green-600 border-b-2 border-green-600 bg-green-50 font-semibold' 
                : 'text-gray-600 hover:text-green-600 hover:bg-green-50/50'
            }`}
          >
            黄历信息
          </button>
          <button 
            onClick={() => setActiveTab('month')}
            className={`flex-1 py-4 text-center font-medium transition-all duration-300 ${
              activeTab === 'month' 
                ? 'text-green-600 border-b-2 border-green-600 bg-green-50 font-semibold' 
                : 'text-gray-600 hover:text-green-600 hover:bg-green-50/50'
            }`}
          >
            万年历
          </button>
        </div>

        {loading ? (
          <div className="flex justify-center items-center h-96 bg-white rounded-b-lg shadow">
            <Spinner size="large" />
          </div>
        ) : error ? (
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded m-4 flex items-center gap-2">
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span>{error}</span>
          </div>
        ) : (
          <>
            {activeTab === 'day' ? (
              <div className="bg-white rounded-b-lg shadow-lg">
                {!huangliData && !lunarData ? (
                  <div className="flex flex-col justify-center items-center h-96 p-4">
                    <svg className="w-16 h-16 text-gray-300 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                    </svg>
                    <p className="text-gray-500 text-center">暂无黄历数据</p>
                    <button 
                      onClick={() => {
                        huangliQuery.refetch();
                        lunarQuery.refetch();
                      }}
                      className="mt-4 bg-green-600 text-white px-4 py-2 rounded hover:bg-green-700 transition-colors"
                    >
                      重试
                    </button>
                  </div>
                ) : (
                  <>
                    <div className="relative py-8 px-4">
                    {/* 左右箭头 */}
                    <button
                      onClick={goToPrevDay}
                      className="absolute left-4 top-1/2 -translate-y-1/2 w-12 h-12 flex items-center justify-center text-gray-400 hover:text-green-600 transition-all transform hover:scale-110"
                    >
                      <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
                      </svg>
                    </button>
                    <button
                      onClick={goToNextDay}
                      className="absolute right-4 top-1/2 -translate-y-1/2 w-12 h-12 flex items-center justify-center text-gray-400 hover:text-green-600 transition-all transform hover:scale-110"
                    >
                      <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                      </svg>
                    </button>

                    {/* 大日期数字 */}
                    <div className="text-center">
                      <div className="text-[120px] font-bold text-green-600 leading-none tracking-tight">
                        {currentDay}
                      </div>
                    </div>
                  </div>

                  {/* 信息栏 */}
                  <div className="flex flex-wrap items-center justify-center gap-4 md:gap-8 py-4 border-t border-b border-gray-100">
                    <div className="flex items-center gap-2 text-gray-600 p-2 rounded-lg hover:bg-green-50 transition-colors">
                      <span className="text-2xl">♈</span>
                      <span className="text-sm">{zodiacSign}</span>
                    </div>
                    <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center shadow-sm transition-transform hover:scale-105">
                      <span className="text-3xl">{zodiacIcon}</span>
                    </div>
                    <div className="text-center p-2 rounded-lg hover:bg-green-50 transition-colors">
                      <div className="text-gray-600 text-sm">农历{lunarYear}</div>
                      <div className="text-green-600 font-medium">{dateStr}</div>
                    </div>
                    {/* 生肖冲煞 */}
                    <div className="flex flex-col items-center gap-2 p-3 border-2 border-green-300 rounded-lg bg-white shadow-sm transition-all hover:shadow-md">
                      <div className="flex items-center gap-2">
                        <span className="text-2xl">{huangliData?.chong ? zodiacIcons[huangliData.chong.replace('冲', '')] || '' : ''}</span>
                        <span className="text-xl text-green-600">冲</span>
                        <span className="text-2xl">{huangliData?.sha ? zodiacIcons[huangliData.sha.replace('煞', '')] || '' : ''}</span>
                      </div>
                      <div className="text-green-600 text-sm font-medium text-center">
                        {huangliData?.chong || ''}{huangliData?.sha || ''}
                      </div>
                    </div>
                    <div className="text-center text-gray-600 text-sm p-2 rounded-lg hover:bg-green-50 transition-colors">
                      <div>星期{weekDays[weekDay]}</div>
                    </div>
                  </div>

                  {/* 主要内容区域 */}
                  <div className="p-4">
                    {/* 六曜和值神 */}
                    <div className="grid grid-cols-2 gap-4 mb-4">
                    <div className="text-center p-3 bg-green-50 rounded-lg shadow-sm transition-all hover:shadow-md">
                      <span className="text-green-600 font-medium">六曜</span>
                      <span className="ml-2 text-gray-700">{huangliData?.liuYao || ''}</span>
                    </div>
                    <div className="text-center p-3 bg-green-50 rounded-lg shadow-sm transition-all hover:shadow-md">
                      <span className="text-green-600 font-medium">值神</span>
                      <span className="ml-2 text-gray-700">{huangliData?.zhiShen || ''}</span>
                    </div>
                  </div>

                  {/* 宜忌区域 */}
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
                    {/* 宜 */}
                    <div className="border border-green-200 rounded-lg p-4 bg-green-50/50 shadow-sm transition-all hover:shadow-md">
                      <div className="flex items-center gap-2 mb-3">
                        <span className="w-8 h-8 bg-green-500 text-white rounded-full flex items-center justify-center font-bold">宜</span>
                      </div>
                      <div className="flex flex-wrap gap-2">
                        {huangliData?.yi?.length ? (
                          huangliData.yi.map((item: string, index: number) => (
                            <span key={index} className="text-sm text-gray-700 bg-white px-2 py-1 rounded-full">{item}</span>
                          ))
                        ) : (
                          <span className="text-gray-500 text-sm">诸事不宜</span>
                        )}
                      </div>
                    </div>

                    {/* 忌 */}
                    <div className="border border-red-200 rounded-lg p-4 bg-red-50/50 shadow-sm transition-all hover:shadow-md">
                      <div className="flex items-center gap-2 mb-3">
                        <span className="w-8 h-8 bg-red-500 text-white rounded-full flex items-center justify-center font-bold">忌</span>
                      </div>
                      <div className="flex flex-wrap gap-2">
                        {huangliData?.ji?.length ? (
                          huangliData.ji.map((item: string, index: number) => (
                            <span key={index} className="text-sm text-gray-700 bg-white px-2 py-1 rounded-full">{item}</span>
                          ))
                        ) : (
                          <span className="text-gray-500 text-sm">诸事皆宜</span>
                        )}
                      </div>
                    </div>
                  </div>

                  {/* 中间区域：八卦图和吉神方位 */}
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
                    {/* 左侧：吉神宜趋 */}
                    <div className="border border-green-200 rounded-lg p-4 shadow-sm transition-all hover:shadow-md">
                      <h3 className="text-green-600 font-medium text-center mb-3">吉神宜趋</h3>
                      <p className="text-sm text-gray-700 text-center leading-relaxed">
                        {huangliData?.jiShen?.join(' ') || huangliData?.jieShen || ''}
                      </p>
                    </div>

                    {/* 中间：八卦方位图 */}
                    <div className="flex justify-center">
                      <div className="relative w-64 h-64 shadow-md transition-transform hover:scale-105">
                        {/* 八卦八边形背景 */}
                        <div className="absolute inset-0 bg-green-500" style={{
                          clipPath: 'polygon(25% 0%, 75% 0%, 100% 25%, 100% 75%, 75% 100%, 25% 100%, 0% 75%, 0% 25%)'
                        }}></div>
                        
                        {/* 八卦文字 - 顺时针方向 */}
                        <div className="absolute top-2 left-1/2 -translate-x-1/2 text-white font-bold text-sm">离</div>
                        <div className="absolute top-6 right-6 text-white font-bold text-sm">坤</div>
                        <div className="absolute top-1/2 right-2 -translate-y-1/2 text-white font-bold text-sm">兑</div>
                        <div className="absolute bottom-6 right-6 text-white font-bold text-sm">乾</div>
                        <div className="absolute bottom-2 left-1/2 -translate-x-1/2 text-white font-bold text-sm">坎</div>
                        <div className="absolute bottom-6 left-6 text-white font-bold text-sm">艮</div>
                        <div className="absolute top-1/2 left-2 -translate-y-1/2 text-white font-bold text-sm">震</div>
                        <div className="absolute top-6 left-6 text-white font-bold text-sm">巽</div>
                        
                        {/* 中间圆形区域 */}
                        <div className="absolute inset-10 bg-white rounded-full flex items-center justify-center shadow-inner">
                          {/* 九宫格数字 */}
                          <div className="grid grid-cols-3 gap-4 w-32 h-32">
                            {[4,9,2,3,5,7,8,1,6].map((num, i) => (
                              <div key={i} className="w-8 h-8 bg-black rounded-full flex items-center justify-center text-white font-bold text-sm">
                                {num}
                              </div>
                            ))}
                          </div>
                        </div>
                        
                        {/* 外部方位标记 */}
                        <div className="absolute -top-4 left-1/2 -translate-x-1/2 w-10 h-10 bg-green-600 rounded-full flex items-center justify-center text-white text-xs font-bold shadow-sm">南</div>
                        <div className="absolute -bottom-4 left-1/2 -translate-x-1/2 w-10 h-10 bg-green-600 rounded-full flex items-center justify-center text-white text-xs font-bold shadow-sm">北</div>
                        <div className="absolute -left-4 top-1/2 -translate-y-1/2 w-10 h-10 bg-green-600 rounded-full flex items-center justify-center text-white text-xs font-bold shadow-sm">东</div>
                        <div className="absolute -right-4 top-1/2 -translate-y-1/2 w-10 h-10 bg-green-600 rounded-full flex items-center justify-center text-white text-xs font-bold shadow-sm">西</div>
                        
                        {/* 斜向方位 */}
                        <div className="absolute top-0 right-0 text-green-800 text-xs font-medium">西南</div>
                        <div className="absolute top-0 left-0 text-green-800 text-xs font-medium">东南</div>
                        <div className="absolute bottom-0 right-0 text-green-800 text-xs font-medium">西北</div>
                        <div className="absolute bottom-0 left-0 text-green-800 text-xs font-medium">东北</div>
                      </div>
                    </div>

                    {/* 右侧：凶神宜忌 */}
                    <div className="border border-red-200 rounded-lg p-4 shadow-sm transition-all hover:shadow-md">
                      <h3 className="text-red-600 font-medium text-center mb-3">凶神宜忌</h3>
                      <p className="text-sm text-gray-700 text-center leading-relaxed">
                        {huangliData?.xiongSha?.join(' ') || huangliData?.zhangSong || ''}
                      </p>
                    </div>
                  </div>

                  {/* 吉神方位 */}
                  <div className="border border-green-200 rounded-lg p-4 mb-6 shadow-sm transition-all hover:shadow-md">
                    <h3 className="text-green-600 font-medium text-center mb-4">吉神方位</h3>
                    <div className="grid grid-cols-5 gap-2 text-center">
                      {directions.map((dir, index) => (
                        <div key={index} className="p-2 rounded-lg hover:bg-green-50 transition-colors">
                          <div className="text-gray-600 text-sm">{dir.name}</div>
                          <div className={`font-medium ${dir.color}`}>{dir.direction}</div>
                        </div>
                      ))}
                    </div>
                  </div>

                  {/* 详细信息表格 */}
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
                    {/* 胎神占方 */}
                    <div className="border border-gray-200 rounded-lg p-4 shadow-sm transition-all hover:shadow-md">
                      <h3 className="text-green-600 font-medium text-center mb-2">胎神占方</h3>
                      <p className="text-sm text-gray-700 text-center">
                        {huangliData?.taiShen || ''}
                      </p>
                    </div>

                    {/* 彭祖百忌 */}
                    <div className="border border-gray-200 rounded-lg p-4 shadow-sm transition-all hover:shadow-md">
                      <h3 className="text-green-600 font-medium text-center mb-2">彭祖百忌</h3>
                      <p className="text-xs text-gray-700 text-center leading-relaxed">
                        {huangliData?.pengZhu || ''}
                      </p>
                    </div>

                    {/* 五行日 */}
                    <div className="border border-gray-200 rounded-lg p-4 shadow-sm transition-all hover:shadow-md">
                      <h3 className="text-green-600 font-medium text-center mb-2">五行日</h3>
                      <p className="text-sm text-gray-700 text-center">
                        {huangliData?.dayWuxing || lunarData?.dayNayin || ''}
                      </p>
                    </div>
                  </div>

                  {/* 建除十二神和二十八星宿 */}
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
                    <div className="border border-green-200 rounded-lg p-4 shadow-sm transition-all hover:shadow-md">
                      <h3 className="text-green-600 font-medium mb-3">建除十二神</h3>
                      <div className="grid grid-cols-2 gap-2 text-sm">
                        <div className="text-gray-600">{todayJianChu}日</div>
                        <div className="text-gray-600">{huangliData?.zhiShen || ''}执位</div>
                      </div>
                    </div>
                    <div className="border border-green-200 rounded-lg p-4 shadow-sm transition-all hover:shadow-md">
                      <h3 className="text-green-600 font-medium mb-3">二十八星宿</h3>
                      <div className="text-center">
                        <div className="text-lg font-medium text-gray-700">{todayXingxiu}宿</div>
                        <div className="text-xs text-gray-500">
                          {todayXingxiu ? 
                            `(${['青龙', '明堂', '天刑', '朱雀', '金匮', '天德', '白虎', '玉堂', '天牢', '玄武', '司命', '勾陈'][todayXingxiu.charCodeAt(0) % 12]})` 
                            : ''
                          }
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* 时辰吉凶 */}
                  <div className="border border-gray-200 rounded-lg overflow-hidden shadow-sm">
                    <h3 className="text-green-600 font-medium text-center py-3 bg-green-50">时辰吉凶</h3>
                    <div className="p-4">
                      {/* 天干地址横向显示 */}
                      <div className="overflow-x-auto">
                        <div className="flex space-x-4 pb-2">
                          {huangliData?.hours && huangliData.hours.length > 0 ? (
                            huangliData.hours.map((hour, index) => {
                              const luck = hour.jiXiong;
                              return (
                                <div key={index} className="flex flex-col items-center min-w-[80px] p-2 rounded-lg hover:bg-green-50 transition-colors">
                                  <div className="font-medium">{tianGanDiZhi[index]}</div>
                                  <div className="text-xs text-gray-500 mt-1">{hour.timeRange}</div>
                                  <div className={`font-medium mt-1 ${luck === '吉' ? 'text-green-600' : 'text-red-600'}`}>
                                    {luck}
                                  </div>
                                </div>
                              );
                            })
                          ) : (
                            tianGanDiZhi.map((item, index) => {
                              const luck = getHourLuck(index);
                              return (
                                <div key={index} className="flex flex-col items-center min-w-[80px] p-2 rounded-lg hover:bg-green-50 transition-colors">
                                  <div className="font-medium">{item}</div>
                                  <div className="text-xs text-gray-500 mt-1">{shichenTime[index]}</div>
                                  <div className={`font-medium mt-1 ${luck === '吉' ? 'text-green-600' : 'text-red-600'}`}>
                                    {luck}
                                  </div>
                                </div>
                              );
                            })
                          )}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
                  </>
                )}
              </div>
            ) : (
              /* 万年历视图 */
              <div className="bg-white rounded-b-lg shadow-lg p-4">
                <WannianliView 
                  currentYear={currentYear}
                  currentMonth={currentMonth}
                  selectedDate={selectedDate}
                  onDateSelect={setSelectedDate}
                  onTodayClick={goToToday}
                />
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};

// 万年历组件
interface WannianliViewProps {
  currentYear: number;
  currentMonth: number;
  selectedDate: Date;
  onDateSelect: (date: Date) => void;
  onTodayClick: () => void;
}

const WannianliView = ({ 
  currentYear, 
  currentMonth, 
  selectedDate, 
  onDateSelect,
  onTodayClick 
}: WannianliViewProps) => {
  // 获取月天数
  const getMonthDays = (year: number, month: number): number[] => {
    const daysInMonth = new Date(year, month, 0).getDate();
    return Array.from({ length: daysInMonth }, (_, i) => i + 1);
  };

  const daysInMonth = getMonthDays(currentYear, currentMonth);
  const firstDayOfMonth = new Date(currentYear, currentMonth - 1, 1).getDay();
  const emptySlots = firstDayOfMonth === 0 ? 6 : firstDayOfMonth - 1;

  const weekDays = ['一', '二', '三', '四', '五', '六', '日'];

  // 获取节假日
  const getHoliday = (month: number, day: number, lunarDateStr: string, solarTerm?: string): string => {
    const solarHolidays: Record<string, string> = {
      '1-1': '元旦', '3-8': '妇女节', '5-1': '劳动节', '6-1': '儿童节',
      '8-1': '建军节', '9-10': '教师节', '10-1': '国庆节'
    };
    const lunarHolidays: Record<string, string> = {
      '正月初一': '春节', '正月十五': '元宵节', '五月初五': '端午节',
      '七月初七': '七夕', '八月十五': '中秋节', '九月初九': '重阳节',
      '腊月初八': '腊八节', '腊月廿三': '小年', '腊月三十': '除夕'
    };
    
    if (lunarHolidays[lunarDateStr]) return lunarHolidays[lunarDateStr];
    if (solarHolidays[`${month}-${day}`]) return solarHolidays[`${month}-${day}`];
    return solarTerm || '';
  };

  return (
    <div>
      {/* 月份导航 */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <button 
            onClick={() => onDateSelect(new Date(currentYear, currentMonth - 2, 1))}
            className="p-2 hover:bg-gray-100 rounded-full transition-all transform hover:scale-105"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
          </button>
          <span className="text-xl font-medium">{currentYear}年{currentMonth}月</span>
          <button 
            onClick={() => onDateSelect(new Date(currentYear, currentMonth, 1))}
            className="p-2 hover:bg-gray-100 rounded-full transition-all transform hover:scale-105"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
          </button>
        </div>
        <button 
          onClick={onTodayClick}
          className="bg-green-600 text-white px-4 py-2 rounded-full text-sm hover:bg-green-700 transition-all transform hover:scale-105"
        >
          今天
        </button>
      </div>

      {/* 星期标题 */}
      <div className="grid grid-cols-7 gap-1 mb-2">
        {weekDays.map((day, index) => (
          <div key={index} className={`text-center py-2 font-medium text-sm ${index >= 5 ? 'text-red-500' : 'text-gray-700'}`}>
            {day}
          </div>
        ))}
      </div>

      {/* 日历网格 */}
      <div className="grid grid-cols-7 gap-1">
        {Array.from({ length: emptySlots }).map((_, index) => (
          <div key={`empty-${index}`} className="h-20"></div>
        ))}
        
        {daysInMonth.map((day) => {
          const date = new Date(currentYear, currentMonth - 1, day);
          const isToday = date.toDateString() === new Date().toDateString();
          const isSelected = date.toDateString() === selectedDate.toDateString();
          const lunarData = getLunar(currentYear, currentMonth, day);
          const lunarDayStr = lunarData.dateStr.substring(2);
          const holiday = getHoliday(currentMonth, day, lunarData.dateStr, lunarData.solarTerm);

          return (
            <div 
              key={day}
              className={`h-20 flex flex-col items-center justify-center rounded-lg cursor-pointer transition-all border ${isSelected 
                ? 'bg-green-500 text-white border-green-500' 
                : isToday 
                  ? 'bg-green-100 border-green-300 text-green-700' 
                  : 'hover:bg-gray-50 border-transparent'
              }`}
              onClick={() => onDateSelect(date)}
            >
              <div className={`text-lg font-medium ${isSelected ? '' : ''}`}>
                {day}
              </div>
              {holiday ? (
                <div className={`text-xs ${isSelected ? 'text-green-100' : 'text-red-500'}`}>
                  {holiday}
                </div>
              ) : (
                <div className={`text-xs ${isSelected ? 'text-green-100' : 'text-gray-500'}`}>
                  {lunarData.lunarDate}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
};

export default HuangliPage;
