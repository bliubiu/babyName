'use client';

import { useState, useCallback } from 'react';
import { useNameStore } from '@/lib/store';
import { useToast } from '@/components/Toast';
import { validateSurname } from '@/lib/validation';
import type { FormData } from '@/types';

interface FormErrors {
  surname?: string;
}

interface UseNameFormReturn {
  formData: FormData;
  errors: FormErrors;
  touched: Record<string, boolean>;
  keywords: string;
  setKeywords: (kw: string) => void;
  setFormData: (data: Partial<FormData>) => void;
  handleSurnameChange: (value: string) => void;
  handleBlur: (field: string) => void;
  handleBirthdayConfirm: (data: Partial<FormData>) => void;
  validateSurname: (value: string) => string | undefined;
  resetErrors: () => void;
}

export function useNameForm(): UseNameFormReturn {
  const { formData, setFormData } = useNameStore();
  const { showToast } = useToast();
  
  const [errors, setErrors] = useState<FormErrors>({});
  const [touched, setTouched] = useState<Record<string, boolean>>({});
  const [keywords, setKeywords] = useState('');

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
    keywords,
    setKeywords,
    setFormData,
    handleSurnameChange,
    handleBlur,
    handleBirthdayConfirm,
    validateSurname,
    resetErrors,
  };
}
