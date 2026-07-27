'use client';

import { useState } from 'react';

import { useNameForm } from '@/hooks/useNameForm';
import type { FormData } from '@/types';
import { BirthdayPicker } from './BirthdayPicker';
import { PreferenceSelector } from './PreferenceSelector';
import { RadicalSelector } from './RadicalSelector';
import { Spinner } from './Spinner';

interface NameFormSubmission {
  formData: FormData;
  keywords: string;
  selectedChars?: string[];
}

interface NameFormProps {
  isGenerating: boolean;
  onSubmit: (data: NameFormSubmission) => void;
  sourceClassic: string;
  onSourceClassicChange: (source: string) => void;
}

export function NameForm({ isGenerating, onSubmit, sourceClassic, onSourceClassicChange }: NameFormProps) {
  const {
    formData,
    errors,
    touched,
    keywords,
    setFormData,
    setKeywords,
    handleSurnameChange,
    handleBlur,
    handleBirthdayConfirm,
  } = useNameForm();

  const [showRadicalSelector, setShowRadicalSelector] = useState(false);
  const [selectedChars, setSelectedChars] = useState<string[]>([]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit({
      formData,
      keywords,
      selectedChars: selectedChars.length > 0 ? selectedChars : undefined,
    });
  };

  const handleRadicalToggle = (chars: string[]) => {
    setSelectedChars(chars);
    const charStr = chars.join('');
    if (charStr) {
      const existing = keywords.replace(/包含字[：:][^\s]*/, '').trim();
      setKeywords(existing ? `${existing} 包含字：${charStr}` : `包含字：${charStr}`);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* 姓氏 */}
      <div className="space-y-1.5">
        <label htmlFor="surname-input" className="form-label">姓氏</label>
        <input
          id="surname-input"
          type="text"
          className={`input-field ${touched.surname && errors.surname ? 'border-crimson/60 ring-2 ring-crimson/10' : ''}`}
          placeholder="请输入姓氏"
          value={formData.surname}
          onChange={(e) => handleSurnameChange(e.target.value)}
          onBlur={() => handleBlur('surname')}
          aria-describedby="surname-error"
        />
        {touched.surname && errors.surname && (
          <p id="surname-error" className="text-crimson text-xs mt-1">{errors.surname}</p>
        )}
      </div>

      {/* 性别 */}
      <div className="space-y-1.5">
        <label className="form-label">性别</label>
        <fieldset>
          <legend className="sr-only">性别</legend>
          <div className="flex gap-3">
            {(['male', 'female'] as const).map((g) => (
              <label
                key={g}
                className={`flex-1 flex items-center justify-center gap-2 py-3 rounded-xl border cursor-pointer transition-all duration-200 ${
                  formData.gender === g
                    ? 'border-crimson/40 bg-crimson/5 text-crimson'
                    : 'border-paper/60 bg-white/50 text-ink-light hover:border-crimson/20'
                }`}
              >
                <input
                  id={`gender-${g}`}
                  type="radio"
                  name="gender"
                  value={g}
                  checked={formData.gender === g}
                  onChange={() => setFormData({ gender: g })}
                  className="sr-only"
                />
                <span className="text-lg">{g === 'male' ? '♂' : '♀'}</span>
                <span className="font-medium">{g === 'male' ? '男' : '女'}</span>
              </label>
            ))}
          </div>
        </fieldset>
      </div>

      {/* 生日 */}
      <div className="space-y-1.5">
        <label className="form-label">出生时间</label>
        <BirthdayPicker onConfirm={handleBirthdayConfirm} />
      </div>

      {/* 名字类型 */}
      <div className="space-y-1.5">
        <label className="form-label">名字类型</label>
        <div className="grid grid-cols-2 gap-3">
          {[
            { value: 'double', label: '双字名', hint: '如 周伯通' },
            { value: 'single', label: '单字名', hint: '如 曹操' },
          ].map((opt) => (
            <label
              key={opt.value}
              className={`flex flex-col items-center py-3 px-4 rounded-xl border cursor-pointer transition-all duration-200 ${
                formData.nameType === opt.value
                  ? 'border-crimson/40 bg-crimson/5'
                  : 'border-paper/60 bg-white/50 hover:border-crimson/20'
              }`}
            >
              <input
                type="radio"
                name="nameType"
                value={opt.value}
                checked={formData.nameType === opt.value}
                onChange={() => setFormData({ nameType: opt.value as 'double' | 'single' })}
                className="sr-only"
              />
              <span className={`font-medium ${formData.nameType === opt.value ? 'text-crimson' : 'text-ink-light'}`}>
                {opt.label}
              </span>
              <span className="text-xs text-ink-light/50 mt-0.5">{opt.hint}</span>
            </label>
          ))}
        </div>
      </div>

      {/* 字辈 */}
      <div className="space-y-1.5">
        <label className="form-label" htmlFor="generation-input">
          字辈 <span className="text-ink-light/40 font-normal">（可不填）</span>
        </label>
        <div className="flex gap-2">
          <input
            id="generation-input"
            type="text"
            className="input-field flex-1"
            placeholder="辈分字"
            value={formData.generation}
            onChange={(e) => setFormData({ generation: e.target.value })}
          />
          <select
            id="generation-position"
            className="input-field w-auto min-w-[110px]"
            value={formData.generationPosition}
            onChange={(e) => setFormData({ generationPosition: e.target.value as 'middle' | 'end' })}
            aria-label="字辈位置"
          >
            <option value="middle">固定中间</option>
            <option value="end">固定最后</option>
          </select>
        </div>
      </div>

      {/* 经典来源 + 个性补充 */}
      <PreferenceSelector
        keywords={keywords}
        onKeywordsChange={setKeywords}
        sourceClassic={sourceClassic}
        onSourceClassicChange={onSourceClassicChange}
      />

      {/* 避讳长辈 */}
      <div className="space-y-1.5">
        <label className="form-label" htmlFor="avoid-elder-input">
          避讳长辈 <span className="text-ink-light/40 font-normal">（可不填，多个姓名用逗号分隔）</span>
        </label>
        <input
          id="avoid-elder-input"
          type="text"
          className="input-field"
          placeholder="如：张三，李四（排除同音同形字）"
          value={formData.avoidElderNames}
          onChange={(e) => setFormData({ avoidElderNames: e.target.value })}
        />
      </div>

      {/* 展开偏旁选字 */}
      <div className="flex items-center justify-end">
        <button
          type="button"
          onClick={() => setShowRadicalSelector(!showRadicalSelector)}
          className="text-sm text-jade hover:text-crimson transition-colors flex items-center gap-1.5 py-1"
          aria-expanded={showRadicalSelector}
        >
          {showRadicalSelector ? '收起' : '展开'}偏旁选字
          <svg
            className={`w-3.5 h-3.5 transition-transform duration-200 ${showRadicalSelector ? 'rotate-180' : ''}`}
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            aria-hidden="true"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </button>
      </div>

      {showRadicalSelector && (
        <RadicalSelector
          onSelectChars={handleRadicalToggle}
          selectedChars={selectedChars}
          disabled={isGenerating}
        />
      )}

      {/* 提交按钮 */}
      <div className="pt-4 flex justify-center">
        <button
          type="submit"
          disabled={isGenerating}
          className="btn-seal w-full sm:w-auto text-lg font-bold px-16 py-3.5 tracking-widest"
        >
          {isGenerating ? (
            <div className="flex items-center justify-center gap-2">
              <Spinner size="small" color="white" />
              <span>正在生成</span>
            </div>
          ) : (
            '开始取名'
          )}
        </button>
      </div>
    </form>
  );
}
