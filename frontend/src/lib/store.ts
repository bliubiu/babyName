'use client';

import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import { HistoryRecord, Name } from '@/types';
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
  generationPosition: 'middle' | 'end';
  nameType: 'double' | 'single';
  birthType: 'solar' | 'lunar';
  preferences: string[];
  nameLength: number;
}

interface NameStore {
  formData: FormData;
  generateResult: any | null;
  compareResult: Name[] | null;
  history: HistoryRecord[];
  favorites: FavoriteData[];
  isLoading: boolean;
  error: string | null;
  _hasHydrated: boolean;

  setFormData: (data: Partial<FormData>) => void;
  resetFormData: () => void;
  setGenerateResult: (result: any | null) => void;
  setCompareResult: (result: Name[] | null) => void;
  setHistory: (history: HistoryRecord[]) => void;
  addHistory: (record: HistoryRecord) => void;
  removeHistory: (id: string) => void;
  setFavorites: (favorites: FavoriteData[]) => void;
  addFavorite: (favorite: FavoriteData) => void;
  removeFavorite: (id: string) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  setHasHydrated: (state: boolean) => void;
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
  generationPosition: 'middle',
  nameType: 'double',
  birthType: 'solar',
  preferences: [],
  nameLength: 2,
};

const createMyStore = () => {
  return create<NameStore>()(
    persist(
      (set) => ({
        formData: initialFormData,
        generateResult: null,
        compareResult: null,
        history: [],
        favorites: [],
        isLoading: false,
        error: null,
        _hasHydrated: false,

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

        setHasHydrated: (state) => set({ _hasHydrated: state }),
      }),
      {
        name: 'namemaster-storage',
        storage: createJSONStorage(() => {
          if (typeof window === 'undefined') {
            return {
              getItem: () => null,
              setItem: () => {},
              removeItem: () => {},
            };
          }
          return localStorage;
        }),
        partialize: (state) => ({
          formData: state.formData,
          favorites: state.favorites,
          history: state.history,
        }),
        onRehydrateStorage: () => (state) => {
          state?.setHasHydrated(true);
        },
      }
    )
  );
};

export const useNameStore = createMyStore();
