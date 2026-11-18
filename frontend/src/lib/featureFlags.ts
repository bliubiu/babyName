/**
 * 构建期功能开关（Vite 在打包时静态注入，运行期不可改）。
 *
 * VITE_SHOW_MATERIALS=true 时显示「物料管理」导航入口；默认隐藏。
 * 原因：namer.db 单库分发后不再随装 JSON 源物料，「重新导入物料」无输入源，
 * 对终端用户是无意义且易误导的操作。仅在开发/运维排障时按需开启。
 */
export function showMaterials(): boolean {
  return import.meta.env.VITE_SHOW_MATERIALS === 'true';
}

/**
 * VITE_SHOW_JSON=true 时显示「JSON 美化查看器」导航入口；默认隐藏。
 * 原因：JSON 查看器面向开发/调试场景（查看接口原始返回），对终端用户
 * 无意义且易误导，与物料入口同口径按需开启。
 */
export function showJsonViewer(): boolean {
  return import.meta.env.VITE_SHOW_JSON === 'true';
}
