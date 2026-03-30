import { FavoriteData } from '@/types';

export interface FavoritesResponse {
  success: boolean;
  message?: string;
  data: FavoriteData[];
}