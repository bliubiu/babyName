// 物料管理（桌面端专属，Go 端 desktop.MaterialState / MaterialImportReport 同构）

/** 物料库内状态 */
export interface MaterialState {
  material: string;
  version: string;
  checksum: string;
  row_count: number;
  imported_at: string;
}

/** 单项物料导入结果 */
export interface MaterialImportItem {
  material: string;
  version: string;
  rows: number;
  skipped: boolean;
  error?: string;
}

/** 一次重导的汇总报告 */
export interface MaterialImportReport {
  items: MaterialImportItem[];
  written: number;
  skipped: number;
  failed: number;
  elapsed?: string;
}

/** 物料状态列表信封（沿用 HTTP 时代形状） */
export interface MaterialStatesResponse {
  success: boolean;
  message?: string;
  data: MaterialState[];
}

/** 重导报告信封 */
export interface MaterialImportResponse {
  success: boolean;
  message?: string;
  data: MaterialImportReport;
}
