'use client';

import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import { Name, GenerateResponse, FormData } from '@/types';

type GenerateResult = GenerateResponse['data'];

interface NameStore {
  formData: FormData;
  generateResult: GenerateResult | null;
  compareResult: Name[] | null;
  isLoading: boolean;
  error: string | null;
  _hasHydrated: boolean;

  setFormData: (data: Partial<FormData>) => void;
  resetFormData: () => void;
  setGenerateResult: (result: GenerateResult | null) => void;
  setCompareResult: (result: Name[] | null) => void;
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
  sourceClassic: '',
  avoidElderNames: '',
};

const createMyStore = () => {
  return create<NameStore>()(
    persist(
      (set) => ({
        formData: initialFormData,
        generateResult: null,
        compareResult: null,
        isLoading: false,
        error: null,
        _hasHydrated: false,

        setFormData: (data) => set((state) => ({
          formData: { ...state.formData, ...data }
        })),

        resetFormData: () => set({ formData: initialFormData }),

        setGenerateResult: (result) => set({ generateResult: result }),

        setCompareResult: (result) => set({ compareResult: result }),

        setLoading: (loading) => set({ isLoading: loading }),

        setError: (error) => set({ error }),

        setHasHydrated: (state) => set({ _hasHydrated: state }),
      }),
      {
        name: 'namer-storage',
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
          // 不持久化 generateResult 和 compareResult（大数据对象可能超出 localStorage 配额）
        }),
        version: 1,
        onRehydrateStorage: () => (state) => {
          state?.setHasHydrated(true);
        },
      }
    )
  );
};

export const useNameStore = createMyStore();
