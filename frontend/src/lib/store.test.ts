import { describe, it, expect, beforeEach } from 'vitest';
import { useNameStore } from './store';

describe('useNameStore', () => {
  beforeEach(() => {
    // 每个测试前重置 store 到初始状态
    useNameStore.setState({
      formData: {
        surname: '',
        gender: 'male',
        birthYear: new Date().getFullYear(),
        birthMonth: 1,
        birthDay: 1,
        birthHour: 12,
        birthMinute: 0,
        birthLocation: '',
        generation: '',
        generationPosition: 'middle',
        nameType: 'double',
        birthType: 'solar',
        preferences: [],
        nameLength: 2,
        sourceClassic: '',
        avoidElderNames: '',
        minFrequencyTier: 0,
        maxFrequencyTier: 0,
      },
      generateResult: null,
      compareResult: null,
      isLoading: false,
      error: null,
      _hasHydrated: false,
    });
  });

  it('应初始化为默认值', () => {
    const state = useNameStore.getState();
    expect(state.formData.surname).toBe('');
    expect(state.formData.gender).toBe('male');
    expect(state.isLoading).toBe(false);
    expect(state.error).toBeNull();
  });

  it('setFormData 应合并表单数据', () => {
    useNameStore.getState().setFormData({ surname: '王' });
    expect(useNameStore.getState().formData.surname).toBe('王');
    // 其他字段应保持不变
    expect(useNameStore.getState().formData.gender).toBe('male');
  });

  it('setFormData 应部分更新', () => {
    useNameStore.getState().setFormData({ surname: '李', gender: 'female' });
    const state = useNameStore.getState();
    expect(state.formData.surname).toBe('李');
    expect(state.formData.gender).toBe('female');
  });

  it('resetFormData 应重置到初始值', () => {
    useNameStore.getState().setFormData({ surname: '王', gender: 'female' });
    useNameStore.getState().resetFormData();
    expect(useNameStore.getState().formData.surname).toBe('');
    expect(useNameStore.getState().formData.gender).toBe('male');
  });

  it('setGenerateResult 应设置结果', () => {
    const mockResult = {} as any; // 仅测试状态设置
    useNameStore.getState().setGenerateResult(mockResult);
    expect(useNameStore.getState().generateResult).toBe(mockResult);
  });

  it('setGenerateResult(null) 应清除结果', () => {
    useNameStore.getState().setGenerateResult({} as any);
    useNameStore.getState().setGenerateResult(null);
    expect(useNameStore.getState().generateResult).toBeNull();
  });

  it('setCompareResult 应设置对比结果', () => {
    const mockNames = [{ full_name: '王天' }] as any;
    useNameStore.getState().setCompareResult(mockNames);
    expect(useNameStore.getState().compareResult).toEqual(mockNames);
  });

  it('setLoading 应更新加载状态', () => {
    useNameStore.getState().setLoading(true);
    expect(useNameStore.getState().isLoading).toBe(true);
    useNameStore.getState().setLoading(false);
    expect(useNameStore.getState().isLoading).toBe(false);
  });

  it('setError 应设置错误信息', () => {
    useNameStore.getState().setError('出错了');
    expect(useNameStore.getState().error).toBe('出错了');
  });

  it('setError(null) 应清除错误', () => {
    useNameStore.getState().setError('出错了');
    useNameStore.getState().setError(null);
    expect(useNameStore.getState().error).toBeNull();
  });
});
