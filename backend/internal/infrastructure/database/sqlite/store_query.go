package sqlite

import (
	"strings"

	_ "modernc.org/sqlite"
	"name/internal/domain/hanzi"
	"name/internal/domain/yijing"
	"name/internal/domain/zodiac"
	"name/internal/infrastructure/database"
)

func (s *Store) GetHexagramByNumber(id int) *yijing.Hexagram {
	for i := range s.Hexagrams {
		if s.Hexagrams[i].Number == id {
			// 返回拷贝，避免外部修改污染内部数据 / 并发读写
			h := s.Hexagrams[i]
			return &h
		}
	}
	return nil
}

func (s *Store) GetHexagrams() []yijing.Hexagram {
	result := make([]yijing.Hexagram, len(s.Hexagrams))
	copy(result, s.Hexagrams)
	return result
}

// --- ZodiacStore ---

func (s *Store) GetZodiacByName(name string) *zodiac.Zodiac {
	for i := range s.Zodiacs {
		if s.Zodiacs[i].Name == name {
			// 返回拷贝，避免外部修改污染内部数据 / 并发读写
			z := s.Zodiacs[i]
			return &z
		}
	}
	return nil
}

func (s *Store) GetZodiacs() []zodiac.Zodiac {
	result := make([]zodiac.Zodiac, len(s.Zodiacs))
	copy(result, s.Zodiacs)
	return result
}

// --- HanziStore ---

func (s *Store) GetHanziByChar(char string) *database.Hanzi {
	if h, ok := hanzi.GetHanzi(char); ok {
		return &database.Hanzi{
			Char:    h.Char,
			Pinyin:  h.Pinyin,
			Wuxing:  h.Wuxing,
			Strokes: h.Strokes,
		}
	}
	return nil
}

func (s *Store) GetHanziByWuxing(wuxing string) []*database.Hanzi {
	var result []*database.Hanzi
	for _, h := range hanzi.HanziData {
		if h.Wuxing == wuxing {
			result = append(result, &database.Hanzi{
				Char:    h.Char,
				Pinyin:  h.Pinyin,
				Wuxing:  h.Wuxing,
				Strokes: h.Strokes,
			})
		}
	}
	return result
}

func (s *Store) GetHanziByStrokes(min, max int) []*database.Hanzi {
	var result []*database.Hanzi
	for _, h := range hanzi.HanziData {
		if h.Strokes >= min && h.Strokes <= max {
			result = append(result, &database.Hanzi{
				Char:    h.Char,
				Pinyin:  h.Pinyin,
				Wuxing:  h.Wuxing,
				Strokes: h.Strokes,
			})
		}
	}
	return result
}

func (s *Store) SearchHanzi(keyword string, limit int) []*database.Hanzi {
	if limit <= 0 {
		limit = 50
	}
	var result []*database.Hanzi
	for _, h := range hanzi.HanziData {
		if strings.Contains(h.Char, keyword) || strings.Contains(strings.ToLower(h.Pinyin), strings.ToLower(keyword)) {
			result = append(result, &database.Hanzi{
				Char:    h.Char,
				Pinyin:  h.Pinyin,
				Wuxing:  h.Wuxing,
				Strokes: h.Strokes,
			})
			if len(result) >= limit {
				break
			}
		}
	}
	return result
}

// --- FeedbackStore ---
