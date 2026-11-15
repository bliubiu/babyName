'use client';

import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getMaterialStates, importMaterials } from '@/lib/api';
import { Spinner } from '@/components/Spinner';
import Navigation from '@/components/Navigation';
import { IconRefresh } from '@/components/Icons';
import type { MaterialState, MaterialImportReport } from '@/types/api/materials';

/**
 * 物料管理页（桌面端专属）
 *
 * 定位（ADR-008 方案 B：状态页 + 手动重导）：
 * - 展示 SQLite 物料库内的物料状态（物料名/版本/指纹/行数/导入时间）；
 * - 提供「重新导入」按钮：从装配时确定的物料目录扫描重导（指纹短路幂等）；
 * - 内存存储模式下状态为空、重导不可用，给出可行动提示；
 * - 重导只写 SQLite，运行期内存缓存需重启应用后生效（页面常驻提示）。
 */
const MaterialsPage = () => {
  const queryClient = useQueryClient();
  const [importError, setImportError] = useState<string | null>(null);
  const [importReport, setImportReport] = useState<MaterialImportReport | null>(null);
  const [importSuccess, setImportSuccess] = useState<string | null>(null);

  const statesQuery = useQuery({
    queryKey: ['material-states'],
    queryFn: () => getMaterialStates(),
    staleTime: 0, // 状态页每次进入都拉最新（重导后可见变化）
  });

  const importMutation = useMutation({
    mutationFn: () => importMaterials(),
    onSuccess: (res) => {
      setImportError(null);
      setImportSuccess(`物料导入完成：写入 ${res.data.written} 项、跳过 ${res.data.skipped} 项、失败 ${res.data.failed} 项${res.data.elapsed ? `，耗时 ${res.data.elapsed}` : ''}`);
      setImportReport(res.data);
      // 重导后状态可能变化，强制刷新列表
      queryClient.invalidateQueries({ queryKey: ['material-states'] });
    },
    onError: (err: unknown) => {
      setImportSuccess(null);
      setImportReport(null);
      setImportError(err instanceof Error ? err.message : '物料导入失败，请稍后重试');
    },
  });

  const states: MaterialState[] = statesQuery.data?.data ?? [];
  const isEmptyMode = !statesQuery.isLoading && states.length === 0;

  return (
    <div className="min-h-screen bg-ivory pb-20">
      <Navigation showBackButton showHistory showFavorites />

      <div className="max-w-3xl mx-auto px-4 py-8">
        <header className="mb-6">
          <h1 className="text-2xl md:text-3xl font-serif text-ink tracking-wide">物料管理</h1>
          <p className="mt-2 text-sm text-ink/60 leading-relaxed">
            字库与经典语料的版本状态。物料以 SQLite 为唯一真源，重导按内容指纹短路、幂等安全。
          </p>
        </header>

        {/* 重导提示区（成功/失败） */}
        {importSuccess && (
          <div className="mb-4 rounded-lg bg-jade/10 border border-jade/20 px-4 py-3 text-sm text-jade" role="status">
            {importSuccess}
          </div>
        )}
        {importError && (
          <div className="mb-4 rounded-lg bg-crimson/10 border border-crimson/20 px-4 py-3 text-sm text-crimson" role="alert">
            {importError}
          </div>
        )}

        {/* 内存模式提示 */}
        {isEmptyMode && (
          <div className="mb-4 rounded-lg bg-amber-50 border border-amber-200 px-4 py-3 text-sm text-amber-800">
            当前未使用 SQLite 存储（内存模式），物料库不可用。历史与收藏在本模式下不持久化。
          </div>
        )}

        {/* 状态列表 */}
        {statesQuery.isLoading ? (
          <div className="flex justify-center py-16">
            <Spinner size="large" />
          </div>
        ) : statesQuery.error ? (
          <div className="rounded-lg bg-crimson/10 border border-crimson/20 px-4 py-3 text-sm text-crimson">
            {statesQuery.error instanceof Error ? statesQuery.error.message : '物料状态读取失败，请稍后重试'}
          </div>
        ) : states.length > 0 ? (
          <div className="overflow-x-auto rounded-xl border border-ink/10 bg-white/70 shadow-sm">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-ink/50 border-b border-ink/10">
                  <th className="px-4 py-3 font-medium">物料</th>
                  <th className="px-4 py-3 font-medium">版本</th>
                  <th className="px-4 py-3 font-medium hidden sm:table-cell">指纹</th>
                  <th className="px-4 py-3 font-medium text-right">行数</th>
                  <th className="px-4 py-3 font-medium hidden md:table-cell">导入时间</th>
                </tr>
              </thead>
              <tbody>
                {states.map((s) => (
                  <tr key={s.material} className="border-b border-ink/5 last:border-b-0 hover:bg-crimson/5 transition-colors">
                    <td className="px-4 py-3 font-medium text-ink">{s.material}</td>
                    <td className="px-4 py-3 text-ink/70">{s.version || '—'}</td>
                    <td className="px-4 py-3 text-ink/50 font-mono text-xs hidden sm:table-cell" title={s.checksum}>
                      {s.checksum ? `${s.checksum.slice(0, 8)}…` : '—'}
                    </td>
                    <td className="px-4 py-3 text-right text-ink/70 tabular-nums">{s.row_count.toLocaleString('zh-CN')}</td>
                    <td className="px-4 py-3 text-ink/50 hidden md:table-cell">{s.imported_at || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}

        {/* 重导报告明细 */}
        {importReport && importReport.items.length > 0 && (
          <section className="mt-6">
            <h2 className="text-base font-serif text-ink mb-3">最近一次导入明细</h2>
            <ul className="rounded-xl border border-ink/10 bg-white/70 divide-y divide-ink/5 text-sm">
              {importReport.items.map((item) => (
                <li key={item.material} className="px-4 py-2.5 flex items-center gap-3">
                  <span className="font-medium text-ink">{item.material}</span>
                  {item.error ? (
                    <span className="text-crimson text-xs flex-1">{item.error}</span>
                  ) : item.skipped ? (
                    <span className="text-ink/40 text-xs flex-1">指纹未变，跳过</span>
                  ) : (
                    <span className="text-jade text-xs flex-1">写入 {item.rows.toLocaleString('zh-CN')} 行</span>
                  )}
                </li>
              ))}
            </ul>
          </section>
        )}

        {/* 操作区 */}
        <section className="mt-6">
          <button
            onClick={() => {
              setImportSuccess(null);
              setImportError(null);
              setImportReport(null);
              importMutation.mutate();
            }}
            disabled={importMutation.isPending || statesQuery.isLoading || isEmptyMode}
            className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-jade text-ivory text-sm font-medium hover:bg-jade/90 disabled:opacity-50 disabled:cursor-not-allowed transition-all"
            aria-label="重新导入物料"
          >
            {importMutation.isPending ? <Spinner size="small" /> : <IconRefresh size={16} />}
            {importMutation.isPending ? '导入中…（可能需要数十秒）' : '重新导入物料'}
          </button>
          <p className="mt-2 text-xs text-ink/40 leading-relaxed">
            重导完成后建议重启应用，使运行期内存缓存（诗词索引/字频统计）与库同步。
          </p>
        </section>
      </div>
    </div>
  );
};

export default MaterialsPage;
