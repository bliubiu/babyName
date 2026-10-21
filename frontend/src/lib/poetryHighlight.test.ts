import { describe, it, expect } from 'vitest';
import { splitByMatchedChars } from './poetryHighlight';

describe('splitByMatchedChars', () => {
  it('无命中字时返回单段纯文本', () => {
    const segs = splitByMatchedChars('南有嘉鱼，烝然罩罩。', undefined);
    expect(segs).toEqual([{ text: '南有嘉鱼，烝然罩罩。', hit: false }]);
  });

  it('空命中数组时退化为纯文本', () => {
    const segs = splitByMatchedChars('嘉宾式燕以乐', []);
    expect(segs).toEqual([{ text: '嘉宾式燕以乐', hit: false }]);
  });

  it('空文本返回空数组', () => {
    expect(splitByMatchedChars('', ['嘉'])).toEqual([]);
  });

  it('命中字被单独切出并标记', () => {
    // 「荔嘉」中只有「嘉」出现在这句里。
    // 注意「嘉」在《南有嘉鱼》「嘉宾式燕以乐」中出现两次，两次都应高亮
    // （都真实出现在句中，不是重复渲染同一处）。
    const text = '南有嘉鱼，烝然罩罩。君子有酒，嘉宾式燕以乐。';
    const segs = splitByMatchedChars(text, ['嘉']);
    const hits = segs.filter(s => s.hit).map(s => s.text);
    expect(hits).toEqual(['嘉', '嘉']);
    // 拼接后应还原原句
    expect(segs.map(s => s.text).join('')).toBe(text);
  });

  it('多个命中字分别切出', () => {
    const segs = splitByMatchedChars('薜荔饰而陆离荐兮', ['荔', '离']);
    const hits = segs.filter(s => s.hit).map(s => s.text);
    expect(hits).toContain('荔');
    expect(hits).toContain('离');
    expect(segs.map(s => s.text).join('')).toBe('薜荔饰而陆离荐兮');
  });

  it('不含于原句的命中字被忽略', () => {
    // 「荔」不在这句里，应被忽略，只高亮「嘉」
    const segs = splitByMatchedChars('嘉宾式燕以乐', ['荔', '嘉']);
    const hits = segs.filter(s => s.hit).map(s => s.text);
    expect(hits).toEqual(['嘉']);
    expect(segs.map(s => s.text).join('')).toBe('嘉宾式燕以乐');
  });

  it('同位置多字命中时短字优先', () => {
    // 「嘉」与「嘉宾」都命中同一位置，应切出较短的「嘉」
    const segs = splitByMatchedChars('嘉宾', ['嘉宾', '嘉']);
    expect(segs.filter(s => s.hit).map(s => s.text)).toEqual(['嘉']);
    expect(segs.map(s => s.text).join('')).toBe('嘉宾');
  });

  it('保留换行与标点', () => {
    const text = '第一句｜第二句';
    const segs = splitByMatchedChars(text, ['第']);
    expect(segs.map(s => s.text).join('')).toBe(text);
  });
});
