import '@testing-library/jest-dom/vitest';
import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import RiskList, { overallRiskLevel } from './RiskList';
import { RiskItem } from '@/types';

const pass: RiskItem = { category: '谐音', level: 'pass', detail: '无不良谐音' };
const warn: RiskItem = { category: '多音字', level: 'warn', detail: '「行」为多音字' };
const fail: RiskItem = { category: '生僻字', level: 'fail', detail: '「龘」未被规范字库收录' };

describe('overallRiskLevel', () => {
  it('空列表为 pass（与后端 OverallRiskLevel 空语义一致）', () => {
    expect(overallRiskLevel([])).toBe('pass');
  });
  it('仅 warn 时为 warn', () => {
    expect(overallRiskLevel([pass, warn])).toBe('warn');
  });
  it('存在 fail 时为 fail（最高级别优先）', () => {
    expect(overallRiskLevel([pass, warn, fail])).toBe('fail');
  });
});

describe('RiskList', () => {
  it('空列表显示通过文案', () => {
    render(<RiskList risks={[]} />);
    expect(screen.getByText('风险体检全部通过')).toBeInTheDocument();
    expect(screen.getByText('未发现需要留意的风险项')).toBeInTheDocument();
  });
  it('渲染每项的类别、等级与说明', () => {
    render(<RiskList risks={[warn, fail]} />);
    expect(screen.getByText('多音字 · 提示')).toBeInTheDocument();
    expect(screen.getByText('「行」为多音字')).toBeInTheDocument();
    expect(screen.getByText('生僻字 · 不建议')).toBeInTheDocument();
    expect(screen.getByText('存在不建议使用的风险项')).toBeInTheDocument();
  });
  it('risks 未传（undefined）按空列表处理不崩溃', () => {
    render(<RiskList />);
    expect(screen.getByText('风险体检全部通过')).toBeInTheDocument();
  });
  it('embedded 紧凑形态：无卡片标题、保留汇总与明细', () => {
    render(<RiskList risks={[fail]} embedded />);
    expect(screen.queryByText('风险体检')).not.toBeInTheDocument();
    expect(screen.getByText('存在不建议使用的风险项')).toBeInTheDocument();
    expect(screen.getByText('「龘」未被规范字库收录')).toBeInTheDocument();
  });
});
