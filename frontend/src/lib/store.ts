import { create } from 'zustand';
import { GenerateRequest, GenerateResponse, HistoryRecord, Name } from '@/types';
import { FavoriteData } from './api';

interface FormData {
  surname: string;
  gender: 'male' | 'female';
  birthYear: number;
  birthMonth: number;
  birthDay: number;
  birthHour: number;
  birthMinute: number;
  birthLocation: string;
  generation: string;
}

interface NameStore {
  formData: FormData;
  generateResult: GenerateResponse['data'] | null;
  compareResult: Name[] | null;
  history: HistoryRecord[];
  favorites: FavoriteData[];
  isLoading: boolean;
  error: string | null;
  
  setFormData: (data: Partial<FormData>) => void;
  resetFormData: () => void;
  setGenerateResult: (result: GenerateResponse['data'] | null) => void;
  setCompareResult: (result: Name[] | null) => void;
  setHistory: (history: HistoryRecord[]) => void;
  addHistory: (record: HistoryRecord) => void;
  removeHistory: (id: string) => void;
  setFavorites: (favorites: FavoriteData[]) => void;
  addFavorite: (favorite: FavoriteData) => void;
  removeFavorite: (id: string) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
}

const initialFormData: FormData = {
  surname: '',
  gender: 'male',
  birthYear: new Date().getFullYear(),
  birthMonth: 1,
  birthDay: 1,
  birthHour: 12,
  birthMinute: 0,
  birthLocation: '',
  generation: '',
};

export const useNameStore = create<NameStore>((set) => ({
  formData: initialFormData,
  generateResult: null,
  compareResult: null,
  history: [],
  favorites: [],
  isLoading: false,
  error: null,

  setFormData: (data) => set((state) => ({
    formData: { ...state.formData, ...data }
  })),

  resetFormData: () => set({ formData: initialFormData }),

  setGenerateResult: (result) => set({ generateResult: result }),

  setCompareResult: (result) => set({ compareResult: result }),

  setHistory: (history) => set({ history }),

  addHistory: (record) => set((state) => ({
    history: [record, ...state.history]
  })),

  removeHistory: (id) => set((state) => ({
    history: state.history.filter((item) => item.id !== id)
  })),

  setFavorites: (favorites) => set({ favorites }),

  addFavorite: (favorite) => set((state) => ({
    favorites: [...state.favorites, favorite]
  })),

  removeFavorite: (id) => set((state) => ({
    favorites: state.favorites.filter((item) => item.id !== id)
  })),

  setLoading: (loading) => set({ isLoading: loading }),

  setError: (error) => set({ error }),
}));
