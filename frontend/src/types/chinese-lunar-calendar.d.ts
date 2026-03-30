declare module 'chinese-lunar-calendar' {
  export interface LunarResult {
    lunarMonth: number;
    lunarDate: number;
    isLeap: boolean;
    solarTerm: string;
    lunarYear: string;
    zodiac: string;
    dateStr: string;
  }

  export function getLunar(year: number, month: number, date: number): LunarResult;
}
