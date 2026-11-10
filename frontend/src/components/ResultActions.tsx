'use client';

import { IconFileText, IconRefresh, IconShare } from './Icons';

interface ResultActionsProps {
  onExportPDF: () => void;
  onShare: () => void;
  onRegenerate: () => void;
  exporting?: boolean;
  sharing?: boolean;
  regenerating?: boolean;
}

/**
 * 结果页统一动作区（docs/06 §3.1）：导出 PDF / 分享 / 重新起名。
 * 三个动作都走后端接口，抽成独立组件便于单独测试回调与忙碌态。
 */
export default function ResultActions({
  onExportPDF,
  onShare,
  onRegenerate,
  exporting = false,
  sharing = false,
  regenerating = false,
}: ResultActionsProps) {
  const secondary = 'inline-flex items-center justify-center gap-1.5 min-h-[44px] px-3.5 py-2 rounded-xl text-sm border border-paper/70 bg-warm-white text-ink hover:bg-warm-white/70 transition-colors disabled:opacity-50';

  return (
    <div role="group" aria-label="结果操作" className="flex flex-wrap items-center gap-2">
      <button type="button" onClick={onExportPDF} disabled={exporting} className={secondary}>
        <IconFileText size={16} />
        {exporting ? '导出中…' : '导出 PDF'}
      </button>
      <button type="button" onClick={onShare} disabled={sharing} className={secondary}>
        <IconShare size={16} />
        {sharing ? '生成链接…' : '分享'}
      </button>
      <button
        type="button"
        onClick={onRegenerate}
        disabled={regenerating}
        className="inline-flex items-center justify-center gap-1.5 min-h-[44px] px-4 py-2 rounded-xl text-sm bg-crimson text-white hover:bg-crimson-light transition-colors disabled:opacity-50"
      >
        <IconRefresh size={16} />
        {regenerating ? '重新生成中…' : '重新起名'}
      </button>
    </div>
  );
}
