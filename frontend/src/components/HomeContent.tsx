'use client';

import { useState, lazy, Suspense } from 'react';
import { useRouter } from 'next/navigation';
import { useNameStore } from '@/lib/store';
import { generateNamesWithProgress, saveHistory } from '@/lib/api';
import { useMutation } from '@tanstack/react-query';
import { useToast } from '@/components/Toast';
import { validateBirthTime, validateSurname } from '@/lib/validation';

// 懒加载取组件本体；提交载荷类型单独以 type-only 方式引入，
// 避免为拿一个类型就把整个组件打进首屏包。
const NameForm = lazy(() => import('@/components/NameForm').then(module => ({
  default: module.NameForm
})));
import type { NameFormSubmission } from '@/components/NameForm';

interface FormErrors {
  surname?: string;
}

export default function HomeContent() {
  const router = useRouter();
  const { showToast } = useToast();
  const { formData, setFormData, setGenerateResult } = useNameStore();

  const [touched, setTouched] = useState<Record<string, boolean>>({});
  const [errors, setErrors] = useState<FormErrors>({});
  const [sourceClassic, setSourceClassic] = useState('');
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  // 长任务进度（异步任务轮询）：stage 阶段名 / percent 百分比
  const [progress, setProgress] = useState<{ stage: string; percent: number } | null>(null);

  const { mutateAsync: generateNamesMutate, isPending: isGenerating } = useMutation({
    mutationFn: (payload: Parameters<typeof generateNamesWithProgress>[0]) =>
      generateNamesWithProgress(payload, (stage, percent) => setProgress({ stage, percent })),
    onSuccess: (data) => {
      setProgress(null);
      if (data && data.names) {
        setGenerateResult(data);

        saveHistory({
          surname: formData.surname,
          gender: formData.gender,
          birth_date: `${formData.birthYear}-${formData.birthMonth}-${formData.birthDay}`,
          birth_time: `${formData.birthHour}:${formData.birthMinute.toString().padStart(2, '0')}`,
          birth_location: formData.birthLocation,
          results: data,
        }).catch((err) => {
          console.error('Failed to save history:', err);
        });

        showToast('名字生成成功！', 'success');
        router.push('/result');
      } else {
        const msg = '生成名字失败';
        setErrorMessage(msg);
        showToast(msg, 'error');
      }
    },
    onError: (error) => {
      setProgress(null);
      const errorMsg = error instanceof Error ? error.message : '未知错误';
      setErrorMessage(errorMsg);
      showToast('生成名字时出错，请稍后重试', 'error');
    },
  });

  // 形参不能省：NameForm 把用户填的寓意关键词与偏旁选字从 onSubmit 传出来，
  // 早前这里写成无参函数，导致那些输入全部被丢弃（后端 meaning_keywords 其实支持）。
  const handleSubmit = (submission: NameFormSubmission) => {
    const surnameError = validateSurname(formData.surname);
    if (surnameError) {
      setErrors({ surname: surnameError });
      setTouched({ surname: true });
      showToast(surnameError, 'error');
      return;
    }

    const birthError = validateBirthTime({
      birthYear: formData.birthYear,
      birthMonth: formData.birthMonth,
      birthDay: formData.birthDay,
      birthHour: formData.birthHour,
      birthMinute: formData.birthMinute,
    });
    if (birthError) {
      showToast(birthError, 'error');
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
      // 寓意关键词来自 NameForm 提交载荷（用户在「个性补充」输入 + 偏旁选字）
      meaning_keywords: submission.keywords.split(/[,，\s]+/).filter(Boolean),
      source_classic: sourceClassic || undefined,
      avoid_elder_names: formData.avoidElderNames
        ? formData.avoidElderNames.split(/[,，\s]+/).filter(Boolean)
        : undefined,
      // 人名频率过滤（来自 Chinese-Names-Corpus 语料统计）
      min_frequency_tier: formData.minFrequencyTier || undefined,
      max_frequency_tier: formData.maxFrequencyTier || undefined,
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

        {/* 生成进度（异步任务轮询） */}
        {isGenerating && progress && (
          <div className="mt-5 mb-1 px-1" aria-live="polite">
            <div className="flex justify-between text-xs text-jade/80 mb-1.5">
              <span>{progress.stage || '生成中'}</span>
              <span>{Math.round(progress.percent)}%</span>
            </div>
            <div className="h-2 bg-ink/5 rounded-full overflow-hidden">
              <div
                className="h-full bg-crimson/70 rounded-full transition-all duration-500"
                style={{ width: `${Math.max(4, Math.min(100, progress.percent))}%` }}
              />
            </div>
          </div>
        )}
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