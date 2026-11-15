/**
 * 桌面端绑定层参数契约回归测试（docs/47 S5）
 *
 * 迁移到 Wails 后不再有 HTTP handler 解包 {data:...} 信封，且 Go []byte
 * 经 json.Marshal 实际以 base64 字符串返回。这两处契约差异都会静默降级：
 * 分享创建失败、PDF 导出报「内容为空」。此文件把它们固化为可回归的断言。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';

const mocks = vi.hoisted(() => ({
  EvaluateName: vi.fn(),
  ExploreNames: vi.fn(),
  SubmitGenerateTask: vi.fn(),
  GetGenerateTask: vi.fn(),
  GetHistory: vi.fn(),
  SaveHistory: vi.fn(),
  DeleteHistory: vi.fn(),
  GetFavorites: vi.fn(),
  SaveFavorite: vi.fn(),
  DeleteFavorite: vi.fn(),
  GenerateReportPDF: vi.fn(),
  CreateShare: vi.fn(),
  GetShare: vi.fn(),
  GetHuangli: vi.fn(),
  GetLunarCalendar: vi.fn(),
  GetLunarMonthCalendar: vi.fn(),
  GetCharGroups: vi.fn(),
  GetNameStats: vi.fn(),
  GetTopNames: vi.fn(),
  GetMaterialStates: vi.fn(),
  ImportMaterials: vi.fn(),
}));

vi.mock('../../wailsjs/go/desktop/App', () => mocks);

import { exportReportPDF, createShareSnapshot } from './api';

const PDF_HEADER = '%PDF-1.4 namer';

describe('exportReportPDF 绑定契约', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('传扁平请求体，不套 HTTP 时代的 {data} 信封', async () => {
    mocks.GenerateReportPDF.mockResolvedValue(btoa(PDF_HEADER));
    await exportReportPDF({ surname: '李', names: [{ full_name: '李安定' }] });

    const arg = mocks.GenerateReportPDF.mock.calls[0][0] as Record<string, unknown>;
    expect(arg).toMatchObject({ surname: '李' });
    expect(arg).not.toHaveProperty('data');
    expect(arg.names).toEqual([{ full_name: '李安定' }]);
  });

  it('解码 Wails 返回的 base64 字符串为 PDF 字节', async () => {
    mocks.GenerateReportPDF.mockResolvedValue(btoa(PDF_HEADER));

    const blob = await exportReportPDF({ surname: '李', names: [] });
    expect(blob.type).toBe('application/pdf');
    expect(blob.size).toBe(PDF_HEADER.length);
    expect((await blob.text()).startsWith('%PDF-')).toBe(true);
  });

  it('兼容生成声明里的 Array<number> 形态', async () => {
    mocks.GenerateReportPDF.mockResolvedValue(Array.from(PDF_HEADER, (c) => c.charCodeAt(0)));

    const blob = await exportReportPDF({ surname: '李', names: [] });
    expect(blob.size).toBe(PDF_HEADER.length);
    expect((await blob.text()).startsWith('%PDF-')).toBe(true);
  });

  it('空字节返回明确错误而不是静默下载空文件', async () => {
    mocks.GenerateReportPDF.mockResolvedValue('');
    await expect(exportReportPDF({ surname: '李', names: [] }))
      .rejects.toThrow('PDF 内容为空');
  });
});

describe('createShareSnapshot 绑定契约', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('传扁平快照，不套 {data} 信封（否则后端白名单清空内容）', async () => {
    const id = 'a'.repeat(32);
    mocks.CreateShare.mockResolvedValue({ id, url: `/#/share?id=${id}` });

    const result = await createShareSnapshot({
      surname: '李',
      names: [{ full_name: '李安定' }],
    });

    const arg = mocks.CreateShare.mock.calls[0][0] as Record<string, unknown>;
    expect(arg).toMatchObject({ surname: '李' });
    expect(arg).not.toHaveProperty('data');
    expect(arg.names).toEqual([{ full_name: '李安定' }]);
    expect(result.id).toBe(id);
    expect(result.url).toContain('/#/share?id=');
  });
});
