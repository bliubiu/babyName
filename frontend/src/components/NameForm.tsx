'use client';

import { useState } from 'react';

import { useNameForm } from '@/hooks/useNameForm';
import type { FormData } from '@/types';
import { BirthdayPicker } from './BirthdayPicker';
import { PreferenceSelector } from './PreferenceSelector';
import { RadicalSelector } from './RadicalSelector';
import { Spinner } from './Spinner';

// 提交载荷：keywords 是用户在「个性补充」里手填的寓意关键词；
// 偏旁选字的用字走独立的 selectedChars 结构化通道。
// 必须导出——HomeContent 要靠它接住这些输入，否则用户输入会被静默丢弃。
export interface NameFormSubmission {
  formData: FormData;
  keywords: string;
  selectedChars?: string[];
}

export interface GenerateProgress {
  stage: string;
  percent: number;
}

interface NameFormProps {
  isGenerating: boolean;
  progress: GenerateProgress | null;
  onSubmit: (data: NameFormSubmission) => void;
  sourceClassic: string;
  onSourceClassicChange: (source: string) => void;
}

/**
 * 分区标题。用 section + h2 建立语义层级（页面 h1 为「宝宝起名」），
 * 序号做成朱砂小印章，呼应咨询笺硬纸封的新中式视觉。
 */
function FormSection({
  id, index, title, note, children,
}: {
  id: string;
  index: string;
  title: string;
  note: string;
  children: React.ReactNode;
}) {
  return (
    <section className="home-section" aria-labelledby={`${id}-title`}>
      <div className="home-section-head">
        <span className="home-section-index" aria-hidden="true">{index}</span>
        <h2 id={`${id}-title`} className="home-section-title">{title}</h2>
        <span className="home-section-note">{note}</span>
        <span className="home-section-rule" aria-hidden="true" />
      </div>
      {children}
    </section>
  );
}

/** 带标签的字段容器；label 通过 htmlFor 与控件关联，保证读屏与点击聚焦。 */
function Field({
  id, label, optional, hint, className = '', children,
}: {
  id: string;
  label: string;
  optional?: boolean;
  hint?: string;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div className={`home-field ${className}`}>
      <label htmlFor={id} className="form-label home-field-label">
        {label}
        {optional && <span className="home-optional">选填</span>}
        {hint && <span className="home-field-hint">{hint}</span>}
      </label>
      {children}
    </div>
  );
}

export function NameForm({
  isGenerating, progress, onSubmit, sourceClassic, onSourceClassicChange,
}: NameFormProps) {
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
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [selectedChars, setSelectedChars] = useState<string[]>([]);

  // 高级选项折叠时给出「已设置 N 项」提示，避免用户以为内容丢失；
  // 三项分别对应避讳长辈、粤语评分、偏旁选字。
  const advancedCount =
    (formData.avoidElderNames?.trim() ? 1 : 0) +
    (formData.enableCantonese ? 1 : 0) +
    (selectedChars.length > 0 ? 1 : 0);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit({
      formData,
      keywords,
      selectedChars: selectedChars.length > 0 ? selectedChars : undefined,
    });
  };

  // 偏旁选字只更新 selectedChars 结构化通道，不再往 keywords 里拼
  // "包含字：木木" 文本（docs/29 A9）：
  //   1) 后端 MeaningKeywords 是按词匹配字义/分类的，"包含字：木木" 这种
  //      拼出来的伪关键词匹配不到任何字，用户的选字等于被丢弃；
  //   2) 早前用 `if (charStr)` 守卫，取消最后一个选字时不清 keywords，
  //      令牌残留并被提交。
  // 这里仍做一次兜底清理：万一历史输入框里已残留该令牌（老会话/localStorage
  // 恢复、用户手打），一并清掉，避免脏关键词被提交。
  const handleRadicalToggle = (chars: string[]) => {
    setSelectedChars(chars);
    const cleaned = keywords.replace(/包含字[：:][^\s]*/g, '').trim();
    if (cleaned !== keywords) {
      setKeywords(cleaned);
    }
  };

  return (
    <form id="home-name-form" onSubmit={handleSubmit} className="home-form">
      {/* ---------- 壹 · 必填信息 ---------- */}
      <FormSection id="home-required" index="壹" title="必填信息" note="八字排盘以出生时刻为准">
        <div className="home-grid">
          <Field id="surname-input" label="姓氏" className="md:col-span-2">
            <input
              id="surname-input"
              type="text"
              className={`input-field ${touched.surname && errors.surname ? 'home-input-invalid' : ''}`}
              placeholder="请输入姓氏"
              value={formData.surname}
              onChange={(e) => handleSurnameChange(e.target.value)}
              onBlur={() => handleBlur('surname')}
              aria-describedby="surname-error"
            />
            {touched.surname && errors.surname && (
              <p id="surname-error" className="home-field-error">{errors.surname}</p>
            )}
          </Field>

          <fieldset className="home-field">
            <legend className="form-label home-field-label">性别</legend>
            <div className="flex gap-3">
              {(['male', 'female'] as const).map((g) => (
                <label
                  key={g}
                  className={`home-option flex-1 ${
                    formData.gender === g ? 'home-option-active' : ''
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
                  <span className="text-lg" aria-hidden="true">{g === 'male' ? '♂' : '♀'}</span>
                  <span className="font-medium">{g === 'male' ? '男' : '女'}</span>
                </label>
              ))}
            </div>
          </fieldset>

          <div className="home-field">
            <span className="form-label home-field-label">出生时间</span>
            <BirthdayPicker onConfirm={handleBirthdayConfirm} />
          </div>

          <fieldset className="home-field md:col-span-2">
            <legend className="form-label home-field-label">名字类型</legend>
            <div className="grid grid-cols-2 gap-3">
              {[
                { value: 'double', label: '双字名', hint: '如 周伯通' },
                { value: 'single', label: '单字名', hint: '如 曹操' },
              ].map((opt) => (
                <label
                  key={opt.value}
                  className={`home-option home-option-stack ${
                    formData.nameType === opt.value ? 'home-option-active' : ''
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
                  <span className="font-medium">{opt.label}</span>
                  <span className="home-option-hint">{opt.hint}</span>
                </label>
              ))}
            </div>
          </fieldset>
        </div>
      </FormSection>

      {/* ---------- 贰 · 取名偏好 ---------- */}
      <FormSection id="home-preference" index="贰" title="取名偏好" note="选填 · 用于收窄候选范围">
        <div className="home-grid">
          <Field
            id="generation-input"
            label="字辈"
            optional
            className="md:col-span-2"
          >
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
                onChange={(e) => setFormData({ generationPosition: e.target.value as 'middle' | 'tail' })}
                aria-label="字辈位置"
              >
                <option value="middle">固定中间</option>
                <option value="tail">固定最后</option>
              </select>
            </div>
          </Field>

          <div className="md:col-span-2">
            <PreferenceSelector
              keywords={keywords}
              onKeywordsChange={setKeywords}
              sourceClassic={sourceClassic}
              onSourceClassicChange={onSourceClassicChange}
            />
          </div>
        </div>
      </FormSection>

      {/* ---------- 叁 · 高级选项（默认折叠） ---------- */}
      <section className="home-section home-advanced" aria-labelledby="home-advanced-title">
        <button
          type="button"
          className="home-advanced-toggle"
          onClick={() => setShowAdvanced(!showAdvanced)}
          aria-expanded={showAdvanced}
          aria-controls="home-advanced-body"
        >
          <span className="home-section-index" aria-hidden="true">叁</span>
          <span id="home-advanced-title" className="home-section-title">高级选项</span>
          <span className="home-advanced-summary">
            <span className={advancedCount > 0 ? 'home-advanced-badge' : 'home-section-note'}>
              {advancedCount > 0 ? `已设置 ${advancedCount} 项` : '选填'}
            </span>
            <span className="home-section-note">避讳 · 粤语评分 · 偏旁选字</span>
          </span>
          <svg
            className={`home-advanced-chevron ${showAdvanced ? 'is-open' : ''}`}
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            aria-hidden="true"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        {showAdvanced && (
          <div id="home-advanced-body" className="home-advanced-body">
            <Field
              id="avoid-elder-input"
              label="避讳长辈"
              optional
              hint="多个姓名用逗号分隔"
            >
              <input
                id="avoid-elder-input"
                type="text"
                className="input-field"
                placeholder="如：张三，李四（排除同音同形字）"
                value={formData.avoidElderNames}
                onChange={(e) => setFormData({ avoidElderNames: e.target.value })}
              />
            </Field>

            <div className="home-field">
              <label className="form-label home-field-label" htmlFor="cantonese-toggle">粤语评分</label>
              <div className="home-switch-row">
                <input
                  id="cantonese-toggle"
                  type="checkbox"
                  checked={formData.enableCantonese}
                  onChange={(e) => setFormData({ enableCantonese: e.target.checked })}
                  aria-describedby="cantonese-toggle-hint"
                  className="mt-0.5 w-4 h-4 accent-crimson cursor-pointer flex-shrink-0"
                />
                <div className="flex-1 min-w-0">
                  <span className="text-sm text-ink">开启粤语多维度评分</span>
                  <p id="cantonese-toggle-hint" className="home-helper mt-1">
                    按粤语音律（声调流畅度、声母/韵母冲突）与粤语谐音独立评出「粤语分」，
                    以轻量权重并入综合分影响排序；名字的粤拼标注不受开关影响，始终展示。
                  </p>
                </div>
              </div>
            </div>

            <div className="flex items-center justify-end">
              <button
                type="button"
                onClick={() => setShowRadicalSelector(!showRadicalSelector)}
                className="home-text-action"
                aria-expanded={showRadicalSelector}
              >
                {showRadicalSelector ? '收起' : '展开'}偏旁选字
                <svg
                  className={`home-text-action-chevron ${showRadicalSelector ? 'is-open' : ''}`}
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
          </div>
        )}
      </section>

      {/* ---------- 提交条（桌面吸底，首屏即可发起生成） ---------- */}
      <div className="home-submit-bar">
        {isGenerating && progress && (
          <div className="home-progress" aria-live="polite">
            <div className="home-progress-meta">
              <span>{progress.stage || '生成中'}</span>
              <span>{Math.round(progress.percent)}%</span>
            </div>
            <div className="home-progress-track">
              <div
                className="home-progress-fill"
                style={{ width: `${Math.max(4, Math.min(100, progress.percent))}%` }}
              />
            </div>
          </div>
        )}

        <div className="home-submit-row">
          <button
            type="submit"
            disabled={isGenerating}
            className="btn-seal home-submit"
          >
            {isGenerating ? (
              <span className="flex items-center justify-center gap-2">
                <Spinner size="small" color="white" />
                <span>正在生成</span>
              </span>
            ) : (
              '开始取名'
            )}
          </button>
        </div>
      </div>
    </form>
  );
}
