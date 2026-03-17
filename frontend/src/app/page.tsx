'use client';

import { useState, useMemo } from 'react';
import { useRouter } from 'next/navigation';
import { useNameStore } from '@/lib/store';
import { generateNames, saveHistory } from '@/lib/api';
import { Spinner } from '@/components/Spinner';
import { useToast } from '@/components/Toast';

interface FormErrors {
  surname?: string;
}

export default function Home() {
  const router = useRouter();
  const { showToast } = useToast();
  const { formData, setFormData, setGenerateResult, setLoading, setError, addHistory, isLoading } = useNameStore();

  const [touched, setTouched] = useState<Record<string, boolean>>({});
  const [errors, setErrors] = useState<FormErrors>({});

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

  const handleSurnameChange = (value: string) => {
    setFormData({ surname: value });
    if (touched.surname) {
      setErrors({ surname: validateSurname(value) });
    }
  };

  const handleBlur = (field: string) => {
    setTouched({ ...touched, [field]: true });
    if (field === 'surname') {
      setErrors({ surname: validateSurname(formData.surname) });
    }
  };

  const isFormValid = useMemo(() => {
    return formData.surname && formData.surname.trim() !== '' && !validateSurname(formData.surname);
  }, [formData.surname]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const surnameError = validateSurname(formData.surname);
    if (surnameError) {
      setErrors({ surname: surnameError });
      setTouched({ surname: true });
      showToast(surnameError, 'error');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await generateNames({
        surname: formData.surname,
        gender: formData.gender,
        birth_year: formData.birthYear,
        birth_month: formData.birthMonth,
        birth_day: formData.birthDay,
        birth_hour: formData.birthHour,
        birth_minute: formData.birthMinute,
        birth_location: formData.birthLocation,
        generation: formData.generation,
      });

      if (response.success && response.data) {
        setGenerateResult(response.data);

        await saveHistory({
          surname: formData.surname,
          gender: formData.gender,
          birth_date: `${formData.birthYear}-${formData.birthMonth}-${formData.birthDay}`,
          birth_time: `${formData.birthHour}:${formData.birthMinute.toString().padStart(2, '0')}`,
          birth_location: formData.birthLocation,
          results: response.data,
        });

        showToast('名字生成成功！', 'success');
        router.push('/result');
      } else {
        const msg = response.message || '生成名字失败';
        setError(msg);
        showToast(msg, 'error');
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : '未知错误';
      setError(errorMessage);
      showToast('生成名字时出错，请稍后重试', 'error');
    } finally {
      setLoading(false);
    }
  };

  const months = Array.from({ length: 12 }, (_, i) => i + 1);
  const days = Array.from({ length: 31 }, (_, i) => i + 1);
  const hours = Array.from({ length: 24 }, (_, i) => i);

  return (
    <main className="min-h-screen py-8 md:py-12 px-3 md:px-4">
      <div className="max-w-2xl mx-auto">
        <div className="text-center mb-8 md:mb-12">
          <h1 className="font-serif text-3xl md:text-4xl lg:text-5xl text-ink mb-3 md:mb-4">
            🍼 宝宝起名大师
          </h1>
          <p className="text-teal text-base md:text-lg">
            用东方智慧，为宝宝选个好名字
          </p>
        </div>

        <div className="card shadow-lg">
          <form onSubmit={handleSubmit} className="space-y-5 md:space-y-6">
            <div className="relative">
              <label className="label">姓氏 <span className="text-crimson">*</span></label>
              <input
                type="text"
                className={`input-field pr-10 ${touched.surname && errors.surname ? 'error' : ''} ${touched.surname && !errors.surname && formData.surname ? 'success' : ''}`}
                placeholder="宝宝的姓氏，比如：王"
                value={formData.surname}
                onChange={(e) => handleSurnameChange(e.target.value)}
                onBlur={() => handleBlur('surname')}
                maxLength={4}
              />
              {touched.surname && !errors.surname && formData.surname && (
                <span className="input-success-icon">✓</span>
              )}
              {touched.surname && errors.surname && (
                <p className="input-error-message">{errors.surname}</p>
              )}
            </div>

            <div>
              <label className="label">性别</label>
              <div className="flex gap-4 md:gap-6">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    name="gender"
                    value="male"
                    checked={formData.gender === 'male'}
                    onChange={() => setFormData({ gender: 'male' })}
                    className="w-4 h-4 text-crimson accent-crimson"
                  />
                  <span className="text-ink">👦 男</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    name="gender"
                    value="female"
                    checked={formData.gender === 'female'}
                    onChange={() => setFormData({ gender: 'female' })}
                    className="w-4 h-4 text-crimson accent-crimson"
                  />
                  <span className="text-ink">👧 女</span>
                </label>
              </div>
            </div>

            <div>
              <label className="label">出生日期</label>
              <div className="grid grid-cols-3 gap-2 md:gap-4">
                <select
                  className="input-field"
                  value={formData.birthYear}
                  onChange={(e) => setFormData({ birthYear: parseInt(e.target.value) })}
                >
                  {Array.from({ length: 10 }, (_, i) => new Date().getFullYear() - i).map((year) => (
                    <option key={year} value={year}>{year}年</option>
                  ))}
                </select>
                <select
                  className="input-field"
                  value={formData.birthMonth}
                  onChange={(e) => setFormData({ birthMonth: parseInt(e.target.value) })}
                >
                  {months.map((month) => (
                    <option key={month} value={month}>{month}月</option>
                  ))}
                </select>
                <select
                  className="input-field"
                  value={formData.birthDay}
                  onChange={(e) => setFormData({ birthDay: parseInt(e.target.value) })}
                >
                  {days.map((day) => (
                    <option key={day} value={day}>{day}日</option>
                  ))}
                </select>
              </div>
            </div>

            <div>
              <label className="label">出生时辰</label>
              <select
                className="input-field"
                value={formData.birthHour}
                onChange={(e) => setFormData({ birthHour: parseInt(e.target.value) })}
              >
                {hours.map((hour) => (
                  <option key={hour} value={hour}>{hour}时</option>
                ))}
              </select>
            </div>

            <div>
              <label className="label">出生地点（可选）</label>
              <input
                type="text"
                className="input-field"
                placeholder="比如：北京、上海"
                value={formData.birthLocation}
                onChange={(e) => setFormData({ birthLocation: e.target.value })}
              />
            </div>

            <div>
              <label className="label">辈字（可选）</label>
              <input
                type="text"
                className="input-field"
                placeholder="比如：思、泽、锦"
                value={formData.generation}
                onChange={(e) => setFormData({ generation: e.target.value })}
                maxLength={2}
              />
              <p className="text-xs text-teal mt-1">家族辈分用字，通常为1-2个汉字</p>
            </div>

            <button
              type="submit"
              disabled={isLoading}
              className={`btn-primary w-full flex items-center justify-center gap-2 ${isLoading ? 'opacity-70 cursor-not-allowed' : ''}`}
            >
              {isLoading ? (
                <>
                  <Spinner size="small" color="white" />
                  <span>正在生成...</span>
                </>
              ) : (
                '开始起名'
              )}
            </button>
          </form>
          
          <div className="mt-4 flex justify-center gap-4 text-sm">
            <button
              onClick={() => router.push('/history')}
              className="text-teal hover:text-crimson transition-colors"
            >
              📜 历史记录
            </button>
            <button
              onClick={() => router.push('/favorites')}
              className="text-teal hover:text-crimson transition-colors"
            >
              ❤️ 我的收藏
            </button>
            <button
              onClick={() => router.push('/stat')}
              className="text-teal hover:text-crimson transition-colors"
            >
              📊 重名率查询
            </button>
          </div>
        </div>
      </div>
    </main>
  );
}
