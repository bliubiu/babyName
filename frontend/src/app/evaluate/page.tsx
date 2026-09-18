'use client';

import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { evaluateName } from '@/lib/api';
import { EvaluateRequest, RiskItem } from '@/types';
import { useNameStore } from '@/lib/store';
import { useToast } from '@/components/Toast';
import { validateSurname } from '@/lib/validation';
import Navigation from '@/components/Navigation';

// 风险等级展示配置
const riskLevelStyle: Record<RiskItem['level'], { label: string; cls: string }> = {
  pass: { label: '通过', cls: 'bg-jade/10 text-jade border-jade/20' },
  warn: { label: '提示', cls: 'bg-gold/10 text-gold border-gold/30' },
  fail: { label: '不建议', cls: 'bg-red-50 text-crimson border-crimson/20' },
};

const riskLevelSummary: Record<'pass' | 'warn' | 'fail', { label: string; cls: string }> = {
  pass: { label: '风险体检全部通过', cls: 'text-jade' },
  warn: { label: '有需要斟酌的提示项', cls: 'text-gold' },
  fail: { label: '存在不建议使用的风险项', cls: 'text-crimson' },
};

export default function EvaluatePage() {
  const { showToast } = useToast();
  const { formData } = useNameStore();

  const [surname, setSurname] = useState('');
  const [givenName, setGivenName] = useState('');
  const [gender, setGender] = useState<'male' | 'female'>('male');
  const [birthYear, setBirthYear] = useState(formData.birthYear || 2024);
  const [birthMonth, setBirthMonth] = useState(1);
  const [birthDay, setBirthDay] = useState(15);
  const [birthHour, setBirthHour] = useState(12);
  const [birthMinute, setBirthMinute] = useState(0);

  const mutation = useMutation({
    mutationFn: (payload: EvaluateRequest) => evaluateName(payload),
    onError: (error) => {
      showToast(error instanceof Error ? error.message : '测名失败，请稍后重试', 'error');
    },
  });

  const handleSubmit = () => {
    const surnameError = validateSurname(surname);
    if (surnameError) {
      showToast(surnameError, 'error');
      return;
    }
    const given = givenName.trim();
    if (!given || given.length > 2) {
      showToast('名字需为 1-2 个汉字', 'error');
      return;
    }
    mutation.mutate({
      surname: surname.trim(),
      given_name: given,
      gender,
      birth_year: birthYear,
      birth_month: birthMonth,
      birth_day: birthDay,
      birth_hour: birthHour,
      birth_minute: birthMinute,
    });
  };

  const result = mutation.data;

  return (
    <main className="min-h-screen py-4 md:py-8 px-4 md:px-6">
      <div className="max-w-2xl mx-auto">
        <Navigation showBackButton={true} showHistory={false} showFavorites={false} />

        {/* 表单卡片 */}
        <div className="consultation-sheet red-ribbon corner-decor mb-6">
          <div className="text-center pt-4 pb-6">
            <h1 className="font-serif text-2xl md:text-3xl text-crimson tracking-widest">
              测名
            </h1>
            <p className="text-sm text-jade/80 tracking-wider mt-3">
              输入候选名字与生辰，看它在八字八维下的表现与风险体检
            </p>
          </div>

          <div className="px-6 pb-6 space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm text-ink-light mb-1.5" htmlFor="eval-surname">姓氏</label>
                <input
                  id="eval-surname"
                  value={surname}
                  onChange={(e) => setSurname(e.target.value)}
                  placeholder="如：王"
                  maxLength={2}
                  className="w-full px-3 py-2 rounded-xl bg-warm-white/60 border border-ink/10 focus:border-crimson/40 outline-none text-ink"
                />
              </div>
              <div>
                <label className="block text-sm text-ink-light mb-1.5" htmlFor="eval-given">名（1-2 字）</label>
                <input
                  id="eval-given"
                  value={givenName}
                  onChange={(e) => setGivenName(e.target.value)}
                  placeholder="如：浩然"
                  maxLength={2}
                  className="w-full px-3 py-2 rounded-xl bg-warm-white/60 border border-ink/10 focus:border-crimson/40 outline-none text-ink"
                />
              </div>
            </div>

            <div>
              <span className="block text-sm text-ink-light mb-1.5">性别</span>
              <div className="flex gap-2">
                {(['male', 'female'] as const).map((g) => (
                  <button
                    key={g}
                    onClick={() => setGender(g)}
                    className={`px-4 py-1.5 rounded-xl text-sm transition-all duration-200 ${
                      gender === g ? 'bg-crimson text-white' : 'bg-ink/5 text-ink hover:bg-ink/10'
                    }`}
                  >
                    {g === 'male' ? '男宝' : '女宝'}
                  </button>
                ))}
              </div>
            </div>

            <div className="grid grid-cols-2 sm:grid-cols-6 gap-3">
              {([
                ['出生年', birthYear, setBirthYear, 1900, 2100],
                ['月', birthMonth, setBirthMonth, 1, 12],
                ['日', birthDay, setBirthDay, 1, 31],
                ['时', birthHour, setBirthHour, 0, 23],
                ['分', birthMinute, setBirthMinute, 0, 59],
              ] as const).map(([label, value, setter, min, max], index) => (
                <div key={label} className={index === 0 ? 'col-span-2 min-w-[7rem]' : ''}>
                  <label className="block text-sm text-ink-light mb-1.5" htmlFor={`eval-${label}`}>{label}</label>
                  <input
                    id={`eval-${label}`}
                    type="number"
                    min={min}
                    max={max}
                    value={value}
                    onChange={(e) => setter(parseInt(e.target.value, 10) || 0)}
                    className="w-full px-3 py-2 rounded-xl bg-warm-white/60 border border-ink/10 focus:border-crimson/40 outline-none text-ink"
                  />
                </div>
              ))}
            </div>

            <button
              onClick={handleSubmit}
              disabled={mutation.isPending}
              className="w-full py-3 bg-crimson text-white rounded-xl font-serif tracking-widest text-lg hover:bg-crimson-light transition-all duration-200 disabled:opacity-50"
            >
              {mutation.isPending ? '测名中…' : '开始测名'}
            </button>
          </div>
        </div>

        {/* 结果 */}
        {result && (
          <div className="space-y-6 animate-fade-in-up">
            {/* 名帖 + 总分 */}
            <div className="consultation-sheet corner-decor py-6 px-6 text-center">
              <div className="text-3xl md:text-4xl text-ink font-serif tracking-widest">
                <span className="text-crimson">{result.name.surname}</span>
                <span>{result.name.given_name}</span>
              </div>
              <div className="text-jade/70 text-sm mt-2">{result.name.pinyin}</div>
              <div className="text-gold text-2xl font-medium mt-3">
                {(result.name.total_score ?? result.name.score).toFixed(1)} 分
              </div>
              <div className="text-ink-light/50 text-xs mt-1">
                {result.name.wuxing} · {result.name.strokes} 画 · 生肖{result.zodiac}
              </div>
              {result.name.meaning && (
                <p className="text-sm text-ink-light mt-4 leading-relaxed">{result.name.meaning}</p>
              )}
            </div>

            {/* 风险体检 */}
            <div className="consultation-sheet corner-decor py-5 px-6">
              <h2 className="section-title mb-4">风险体检</h2>
              <p className={`text-sm mb-4 ${riskLevelSummary[result.risk_level].cls}`}>
                {riskLevelSummary[result.risk_level].label}
              </p>
              <ul className="space-y-2.5">
                {result.risks.map((item, idx) => (
                  <li key={idx} className="flex items-start gap-3">
                    <span className={`shrink-0 px-2 py-0.5 rounded-lg text-xs border ${riskLevelStyle[item.level].cls}`}>
                      {item.category} · {riskLevelStyle[item.level].label}
                    </span>
                    <span className="text-sm text-ink-light leading-relaxed">{item.detail}</span>
                  </li>
                ))}
              </ul>
            </div>

            {/* 八维评分明细（score_detail 与结果页同源） */}
            {result.name.score_detail && result.name.score_detail.length > 0 && (
              <div className="consultation-sheet corner-decor py-5 px-6">
                <h2 className="section-title mb-4">八维评分依据</h2>
                <div className="space-y-4">
                  {result.name.score_detail.map((dim) => (
                    <div key={dim.name}>
                      <div className="flex justify-between text-sm mb-1">
                        <span className="text-ink">{dim.name}</span>
                        <span className="text-jade font-medium">{dim.score.toFixed(1)}</span>
                      </div>
                      <div className="h-1.5 bg-ink/5 rounded-full overflow-hidden">
                        <div
                          className="h-full bg-jade/60 rounded-full"
                          style={{ width: `${Math.max(2, Math.min(100, dim.score))}%` }}
                        />
                      </div>
                      {dim.detail && (
                        <p className="text-xs text-ink-light/70 mt-1 leading-relaxed">{dim.detail}</p>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* 诗词出处 */}
            {result.name.poetry_sentence && (
              <div className="consultation-sheet corner-decor py-5 px-6">
                <h2 className="section-title mb-3">诗词出处</h2>
                {result.name.poetry_source && (
                  <p className="text-sm text-jade mb-2">
                    《{result.name.poetry_source}》{result.name.poetry_chapter}
                    {result.name.poetry_author ? ` · ${result.name.poetry_dynasty} ${result.name.poetry_author}` : ''}
                  </p>
                )}
                <p className="font-serif text-ink leading-loose">{result.name.poetry_sentence}</p>
              </div>
            )}
          </div>
        )}
      </div>
    </main>
  );
}
