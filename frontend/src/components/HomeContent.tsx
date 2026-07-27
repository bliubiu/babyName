'use client';

import { useState, lazy, Suspense } from 'react';
import { useRouter } from 'next/navigation';
import { useNameStore } from '@/lib/store';
import { generateNames, saveHistory } from '@/lib/api';
import { useMutation } from '@tanstack/react-query';
import { useToast } from '@/components/Toast';
import { validateSurname } from '@/lib/validation';

const NameForm = lazy(() => import('@/components/NameForm').then(module => ({
  default: module.NameForm
})));

interface FormErrors {
  surname?: string;
}

export default function HomeContent() {
  const router = useRouter();
  const { showToast } = useToast();
  const { formData, setFormData, setGenerateResult } = useNameStore();

  const [touched, setTouched] = useState<Record<string, boolean>>({});
  const [errors, setErrors] = useState<FormErrors>({});
  const [keywords, setKeywords] = useState('');
  const [sourceClassic, setSourceClassic] = useState('');
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

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
      meaning_keywords: keywords.split(/[,，\s]+/).filter(Boolean),
      source_classic: sourceClassic || undefined,
      avoid_elder_names: formData.avoidElderNames
        ? formData.avoidElderNames.split(/[,，\s]+/).filter(Boolean)
        : undefined,
    };

    generateNamesMutate(formDataForApi);
  };

  return (
    <div className="max-w-2xl mx-auto relative z-10">
      {/* 主卡片 */}
      <div className="consultation-sheet red-ribbon corner-decor ink-wash-bg">
        {/* 标题区域 */}
        <div className="text-center pt-4 pb-6">
          <h1 className="font-serif text-3xl md:text-4xl text-crimson tracking-widest ink-calligraphy brush-underline">
            宝宝起名
          </h1>
          <div className="flex items-center justify-center gap-4 mt-5 mb-3">
            <div className="brush-divider max-w-[72px] flex-1" />
            <span className="text-paper-edge/50 text-xs tracking-[0.6em] font-serif">如意</span>
            <div className="brush-divider max-w-[72px] flex-1" />
          </div>
          <p className="text-sm text-jade/80 tracking-wider">
            输入宝宝信息，为您推荐最合适的名字
          </p>
        </div>

        {/* 错误提示 */}
        {errorMessage && (
          <div className="mb-5 p-3 bg-red-50/80 border border-crimson/15 rounded-xl text-crimson text-sm">
            {errorMessage}
          </div>
        )}

        {/* 表单 */}
        <Suspense fallback={
          <div className="flex items-center justify-center py-12">
            <div className="text-jade/60 text-sm">加载中...</div>
          </div>
        }>
          <NameForm
            isGenerating={isGenerating}
            onSubmit={handleSubmit}
            sourceClassic={sourceClassic}
            onSourceClassicChange={setSourceClassic}
          />
        </Suspense>
      </div>

      {/* 底部装饰 */}
      <div className="text-center mt-8 animate-fade-in-up" style={{ animationDelay: '0.6s' }}>
        <p className="text-ink-light/30 text-xs tracking-wider">
          基于八字五行 · 易经卦象 · 生肖属相
        </p>
      </div>
    </div>
  );
}