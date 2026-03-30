'use client';

import { useNameForm } from '@/hooks/useNameForm';
import { BirthdayPicker } from './BirthdayPicker';
import { PreferenceSelector } from './PreferenceSelector';
import { Spinner } from './Spinner';

interface NameFormProps {
  isGenerating: boolean;
  onSubmit: (formData: any) => void;
}

export function NameForm({ isGenerating, onSubmit }: NameFormProps) {
  const {
    formData,
    errors,
    touched,
    preferences,
    keywords,
    setFormData,
    setPreferences,
    setKeywords,
    handleSurnameChange,
    handleBlur,
    handlePreferenceToggle,
    handleBirthdayConfirm,
  } = useNameForm();
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

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit({
      formData,
      preferences,
      keywords,
    });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-4">
        <label htmlFor="surname-input" className="w-20 text-right text-stone-700 tracking-wider sm:block hidden">姓 氏:</label>
        <label htmlFor="surname-input" className="text-stone-700 tracking-wider sm:hidden">姓 氏:</label>
        <input
          id="surname-input"
          type="text"
          className={`proto-input flex-1 max-w-xs px-4 py-2 border border-stone-300 bg-white focus:outline-none focus:border-red-500 ${touched.surname && errors.surname ? 'border-red-500' : ''}`}
          placeholder="请输入姓氏"
          value={formData.surname}
          onChange={(e) => handleSurnameChange(e.target.value)}
          onBlur={() => handleBlur('surname')}
          aria-describedby="surname-error"
        />
      </div>
      {touched.surname && errors.surname && (
        <p id="surname-error" className="input-error-message ml-0 sm:ml-20">{errors.surname}</p>
      )}

      <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-4">
        <label htmlFor="gender-male" className="w-20 text-right text-stone-700 tracking-wider sm:block hidden">性 别:</label>
        <label htmlFor="gender-male" className="text-stone-700 tracking-wider sm:hidden">性 别:</label>
        <fieldset>
          <legend className="sr-only">性别</legend>
          <div className="flex gap-6">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                id="gender-male"
                type="radio"
                name="gender"
                value="male"
                checked={formData.gender === 'male'}
                onChange={() => setFormData({ gender: 'male' })}
                className="w-4 h-4 accent-blue-600"
                aria-label="男性"
              />
              <span className="text-stone-700">男</span>
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                id="gender-female"
                type="radio"
                name="gender"
                value="female"
                checked={formData.gender === 'female'}
                onChange={() => setFormData({ gender: 'female' })}
                className="w-4 h-4 accent-blue-600"
                aria-label="女性"
              />
              <span className="text-stone-700">女</span>
            </label>
          </div>
        </fieldset>
      </div>

      <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-4">
        <label className="w-20 text-right text-stone-700 tracking-wider sm:block hidden">生 日:</label>
        <label className="text-stone-700 tracking-wider sm:hidden">生 日:</label>
        <BirthdayPicker onConfirm={handleBirthdayConfirm} />
      </div>

      <div className="flex flex-col sm:flex-row sm:items-start gap-2 sm:gap-4">
        <label className="w-20 text-right text-stone-700 text-sm sm:block hidden">名字类型:</label>
        <label className="text-stone-700 text-sm sm:hidden">名字类型:</label>
        <div className="flex flex-col sm:flex-row gap-3 sm:gap-6">
          <label className="flex items-center gap-2 cursor-pointer">
            <input type="radio" name="nameType" value="double" checked={formData.nameType === 'double'} onChange={() => setFormData({ nameType: 'double' })} className="w-4 h-4 accent-blue-600" />
            <span className="text-stone-700">双字名 <span className="text-stone-400 text-sm">(如: 周伯通)</span></span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input type="radio" name="nameType" value="single" checked={formData.nameType === 'single'} onChange={() => setFormData({ nameType: 'single' })} className="w-4 h-4 accent-blue-600" />
            <span className="text-stone-700">单字名 <span className="text-stone-400 text-sm">(如: 曹操)</span></span>
          </label>
        </div>
      </div>

      <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-4">
        <label className="w-20 text-right text-red-600 tracking-wider sm:block hidden">字辈:</label>
        <label className="text-red-600 tracking-wider sm:hidden">字辈:</label>
        <div className="flex flex-wrap items-center gap-2">
          <input
            type="text"
            className="w-20 px-4 py-2 border border-stone-300 bg-white focus:outline-none focus:border-red-500"
            placeholder=""
            value={formData.generation}
            onChange={(e) => setFormData({ generation: e.target.value })}
          />
          <select
            className="px-3 py-2 border border-stone-300 bg-white focus:outline-none focus:border-red-500"
            value={formData.generationPosition}
            onChange={(e) => setFormData({ generationPosition: e.target.value as 'middle' | 'end' })}
          >
            <option value="middle">固定中间</option>
            <option value="end">固定最后</option>
          </select>
          <span className="text-blue-500 text-sm">可以不填写</span>
        </div>
      </div>

      <PreferenceSelector
        preferences={preferences}
        onToggle={handlePreferenceToggle}
        keywords={keywords}
        onKeywordsChange={setKeywords}
      />

      <div className="pt-6 flex justify-center">
        <button
          type="submit"
          disabled={isGenerating}
          className={`bg-gradient-to-r from-red-700 to-red-800 text-white text-lg sm:text-xl font-bold rounded-lg shadow-lg hover:from-red-800 hover:to-red-900 transition-all transform hover:scale-105 px-12 sm:px-16 py-3 sm:py-4 w-full sm:w-auto ${isGenerating ? 'opacity-70 cursor-not-allowed' : ''}`}
        >
          {isGenerating ? (
            <div className="flex items-center justify-center gap-2">
              <Spinner size="small" color="white" />
              <span>正在生成...</span>
            </div>
          ) : (
            '开始取名'
          )}
        </button>
      </div>
    </form>
  );
}
