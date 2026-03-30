'use client';

import { useState, lazy, Suspense } from 'react';
import { useRouter } from 'next/navigation';
import { useNameStore } from '@/lib/store';
import { generateNames, saveHistory } from '@/lib/api';
import { useMutation } from '@tanstack/react-query';
import { useToast } from '@/components/Toast';
import Navigation from '@/components/Navigation';

// 懒加载大型组件
const NameForm = lazy(() => import('@/components/NameForm').then(module => ({
  default: module.NameForm
})));

interface FormErrors {
  surname?: string;
}

export default function Home() {
  const router = useRouter();
  const { showToast } = useToast();
  const { formData, setFormData, setGenerateResult } = useNameStore();

  const [touched, setTouched] = useState<Record<string, boolean>>({});
  const [errors, setErrors] = useState<FormErrors>({});
  const [preferences, setPreferences] = useState<string[]>([]);
  const [keywords, setKeywords] = useState('');
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const validateSurname = (value: string): string | undefined => {
    if (!value || value.trim() === '') {
      return '请输入姓氏';
    }
    if (value.length > 4) {
      return '姓氏不能超过4个字符';
    }
    if (!/^[\u4e00-\u9fa5]+$/.test(value)) {
      return '姓氏只能包含中文';
    }
    return undefined;
  };

  const { mutateAsync: generateNamesMutate, isPending: isGenerating } = useMutation({
    mutationFn: generateNames,
    onSuccess: (response) => {
      if (response.success && response.data) {
        setGenerateResult(response.data);

        saveHistory({
          surname: formData.surname,
          gender: formData.gender,
          birth_date: `${formData.birthYear}-${formData.birthMonth}-${formData.birthDay}`,
          birth_time: `${formData.birthHour}:${formData.birthMinute.toString().padStart(2, '0')}`,
          birth_location: formData.birthLocation,
          results: response.data,
        }).catch((err) => {
          console.error('Failed to save history:', err);
        });

        showToast('名字生成成功！', 'success');
        router.push('/result');
      } else {
        const msg = response.message || '生成名字失败';
        setErrorMessage(msg);
        showToast(msg, 'error');
      }
    },
    onError: (error) => {
      const errorMsg = error instanceof Error ? error.message : '未知错误';
      setErrorMessage(errorMsg);
      showToast('生成名字时出错，请稍后重试', 'error');
    },
  });

  const handleSubmit = () => {
    const surnameError = validateSurname(formData.surname);
    if (surnameError) {
      setErrors({ surname: surnameError });
      setTouched({ surname: true });
      showToast(surnameError, 'error');
      return;
    }

    setErrorMessage(null);

    const formDataForApi = {
      surname: formData.surname,
      gender: formData.gender,
      birth_year: formData.birthYear,
      birth_month: formData.birthMonth,
      birth_day: formData.birthDay,
      birth_hour: formData.birthHour,
      birth_minute: formData.birthMinute,
      birth_location: formData.birthLocation,
      generation: formData.generation,
      generation_position: formData.generationPosition,
      name_type: formData.nameType,
      name_length: formData.nameType === 'double' ? 2 : 1,
      preferences: preferences,
      keywords: keywords,
    };

    generateNamesMutate(formDataForApi);
  };

  return (
    <main className="min-h-screen py-8 md:py-12 px-3 md:px-4">
      <div className="max-w-4xl mx-auto">
        <Navigation showBackButton={false} showHistory={true} showFavorites={true} />
        <div className="relative proto-card shadow-lg p-6 md:p-8 lg:p-10">
          <div className="absolute top-2 left-2 corner-decoration corner-tl"></div>
          <div className="absolute top-2 right-2 corner-decoration corner-tr"></div>
          <div className="absolute bottom-2 left-2 corner-decoration corner-bl"></div>
          <div className="absolute bottom-2 right-2 corner-decoration corner-br"></div>

          <div className="relative">
            <div className="absolute top-0 left-0 -translate-y-1/2 bg-[#B91C1C] text-white px-4 py-1 border-2 border-[#D97706] rounded">
              <div className="text-lg font-bold">必填信息</div>
            </div>

            {errorMessage && (
              <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
                {errorMessage}
              </div>
            )}

            <div className="pt-6 space-y-5">
              <Suspense fallback={
                <div className="flex items-center justify-center py-12">
                  <div className="text-stone-600">加载中...</div>
                </div>
              }>
                <NameForm
                  isGenerating={isGenerating}
                  onSubmit={handleSubmit}
                />
              </Suspense>
            </div>
          </div>
        </div>
      </div>
    </main>
  );
}
