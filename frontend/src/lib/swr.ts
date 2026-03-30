import useSWR from 'swr';
import { generateNames, getHistory, saveHistory, deleteHistory, getHexagrams, getZodiacs, getFavorites, saveFavorite, deleteFavorite, checkFavorite, getNameStats, getProvinceStats, getTopNames, FavoriteData } from './api';

const fetcher = async (url: string) => {
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }
  return response.json();
};

export function useHistory() {
  return useSWR('/api/v1/history', fetcher);
}

export function useFavorites() {
  return useSWR('/api/v1/favorites', fetcher);
}

export function useHexagrams() {
  return useSWR('/api/v1/yijing/hexagram', fetcher);
}

export function useZodiacs() {
  return useSWR('/api/v1/zodiac', fetcher);
}

export function useNameStats(name: string) {
  return useSWR(name ? `/api/v1/namestat/${encodeURIComponent(name)}` : null, fetcher);
}

export function useTopNames(limit: number = 20) {
  return useSWR(`/api/v1/namestat?limit=${limit}`, fetcher);
}

// 保留原有的API函数用于手动调用
export { generateNames, saveHistory, deleteHistory, saveFavorite, deleteFavorite, checkFavorite, getProvinceStats };

// 导出类型
export type { FavoriteData } from './api';
