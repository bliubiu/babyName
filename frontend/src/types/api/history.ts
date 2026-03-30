import { GenerateResponse } from '@/types';

export interface HistoryResponse {
  success: boolean;
  message?: string;
  data: Array<{
    id: string;
    surname: string;
    gender: 'male' | 'female';
    birth_date: string;
    birth_time: string;
    birth_location: string;
    results: GenerateResponse['data'] | string;
    created_at: string;
  }>;
}