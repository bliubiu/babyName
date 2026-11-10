import type { FormData, GenerateRequest } from '@/types';

/**
 * 生成请求的**单一装配链**。
 *
 * 首页提交与结果页「重新起名」都必须经这里构造请求体：此前表单字段分散在
 * HomeContent 内联对象里，结果页若要复用只能复制一遍，属于本项目反复出现的
 * 「双口径」缺陷模式（装配字段漏传 / 命名不一致）。
 *
 * 额外的表单交互态（寓意关键词、偏旁选字、典籍筛选）由调用方显式传入；
 * 结果页重新生成时没有这些交互输入，留空即可，不会影响核心生辰字段。
 */
export interface GenerateRequestExtras {
  keywords?: string;
  selectedChars?: string[];
  sourceClassic?: string;
}

export function buildGenerateRequest(
  formData: FormData,
  extras: GenerateRequestExtras = {}
): GenerateRequest {
  return {
    surname: formData.surname,
    gender: formData.gender,
    birth_year: formData.birthYear,
    birth_month: formData.birthMonth,
    birth_day: formData.birthDay,
    birth_hour: formData.birthHour,
    birth_minute: formData.birthMinute,
    birth_location: formData.birthLocation,
    generation: formData.generation,
    generation_position: formData.generationPosition,
    name_type: formData.nameType,
    name_length: formData.nameType === 'double' ? 2 : 1,
    // 寓意关键词来自「个性补充」输入；偏旁选字走 selected_chars 结构化通道，
    // 不能混进 keywords（拼成 "包含字：木木" 后端匹配不到任何字）。
    meaning_keywords: (extras.keywords ?? '').split(/[,，\s]+/).filter(Boolean),
    selected_chars: extras.selectedChars,
    source_classic: extras.sourceClassic || formData.sourceClassic || undefined,
    avoid_elder_names: formData.avoidElderNames
      ? formData.avoidElderNames.split(/[,，\s]+/).filter(Boolean)
      : undefined,
    min_frequency_tier: formData.minFrequencyTier || undefined,
    max_frequency_tier: formData.maxFrequencyTier || undefined,
    enable_cantonese: formData.enableCantonese || undefined,
  };
}
