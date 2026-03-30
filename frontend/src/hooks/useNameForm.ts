'use client';

import { useState, useCallback } from 'react';
import { useNameStore } from '@/lib/store';
import { useToast } from '@/components/Toast';

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
}

interface FormErrors {
  surname?: string;
}

interface UseNameFormReturn {
  formData: FormData;
  errors: FormErrors;
  touched: Record<string, boolean>;
  preferences: string[];
  keywords: string;
  setPreferences: (prefs: string[]) => void;
  setKeywords: (kw: string) => void;
  setFormData: (data: Partial<FormData>) => void;
  handleSurnameChange: (value: string) => void;
  handleBlur: (field: string) => void;
  handlePreferenceToggle: (preference: string) => void;
  handleBirthdayConfirm: (data: Partial<FormData>) => void;
  validateSurname: (value: string) => string | undefined;
  resetErrors: () => void;
}

export function useNameForm(): UseNameFormReturn {
  const { formData, setFormData } = useNameStore();
  const { showToast } = useToast();
  
  const [errors, setErrors] = useState<FormErrors>({});
  const [touched, setTouched] = useState<Record<string, boolean>>({});
  const [preferences, setPreferences] = useState<string[]>([]);
  const [keywords, setKeywords] = useState('');

  const validateSurname = useCallback((value: string): string | undefined => {
    if (!value || value.trim() === '') {
      return '请输入姓氏';
    }
    if (value.length > 4) {
      return '姓氏不能超过 4 个字符';
    }
    if (!/^[\u4e00-\u9fa5]+$/.test(value)) {
      return '姓氏只能包含中文';
    }
    return undefined;
  }, []);

  const handleSurnameChange = useCallback((value: string) => {
    setFormData({ surname: value });
    if (touched.surname) {
      setErrors({ surname: validateSurname(value) });
    }
  }, [setFormData, touched.surname, validateSurname]);

  const handleBlur = useCallback((field: string) => {
    setTouched(prev => ({ ...prev, [field]: true }));
    if (field === 'surname') {
      setErrors({ surname: validateSurname(formData.surname) });
    }
  }, [formData.surname, validateSurname]);

  const handlePreferenceToggle = useCallback((preference: string) => {
    setPreferences(prev => 
      prev.includes(preference) 
        ? prev.filter(p => p !== preference)
        : [...prev, preference]
    );
  }, []);

  const handleBirthdayConfirm = useCallback((data: Partial<FormData>) => {
    setFormData(data);
  }, [setFormData]);

  const resetErrors = useCallback(() => {
    setErrors({});
  }, []);

  return {
    formData,
    errors,
    touched,
    preferences,
    keywords,
    setPreferences,
    setKeywords,
    setFormData,
    handleSurnameChange,
    handleBlur,
    handlePreferenceToggle,
    handleBirthdayConfirm,
    validateSurname,
    resetErrors,
  };
}
