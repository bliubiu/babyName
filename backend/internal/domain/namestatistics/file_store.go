package namestatistics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"name/internal/infrastructure/database"
)

// FileNameStatStore 基于 JSON 数据文件的 NameStatStore 实现。
//
// 数据源位于 data 目录（backend/data），由 cmd/build_namestats 从
// Chinese-Names-Corpus 语料生成：
//   - surname_stats.json     姓氏统计（按人数降序）
//   - given_name_stats.json  按姓氏分组名字统计（每姓氏 Top100）
//   - full_name_stats.json   全名统计（Top50000，按人数降序）
//   - name_gender_stats.json 名字（去姓氏）性别分布（按总人数降序）
//   - name_frequency.json    单字频率（meta.total_names 为语料总人数）
//
// 首次查询时懒加载并缓存（静态统计，内存驻留）。
type FileNameStatStore struct {
	dataDir string

	mu         sync.Mutex
	loaded     bool
	surnames   []database.SurnameStat
	givenNames []database.GivenNameStat
	fullNames  []database.FullNameStat
	gender     []database.NameGenderStat
	totalNames int
}

// NewFileNameStatStore 创建基于 JSON 数据的姓名统计存储
func NewFileNameStatStore(dataDir string) database.NameStatStore {
	return &FileNameStatStore{dataDir: dataDir}
}

func (s *FileNameStatStore) ensureLoaded() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loaded {
		return nil
	}

	surnames, err := loadJSONFile[database.SurnameStat](s.dataDir, "surname_stats.json")
	if err != nil {
		return fmt.Errorf("加载姓氏统计失败: %w", err)
	}
	given, err := loadJSONFile[database.GivenNameStat](s.dataDir, "given_name_stats.json")
	if err != nil {
		return fmt.Errorf("加载名字统计失败: %w", err)
	}
	full, err := loadJSONFile[database.FullNameStat](s.dataDir, "full_name_stats.json")
	if err != nil {
		return fmt.Errorf("加载全名统计失败: %w", err)
	}
	gender, err := loadJSONFile[database.NameGenderStat](s.dataDir, "name_gender_stats.json")
	if err != nil {
		return fmt.Errorf("加载名字性别统计失败: %w", err)
	}
	total, err := s.loadTotalNames()
	if err != nil {
		return err
	}

	s.surnames = surnames
	s.givenNames = given
	s.fullNames = full
	s.gender = gender
	s.totalNames = total
	s.loaded = true
	return nil
}

// loadTotalNames 从 name_frequency.json 的 meta.total_names 读取语料总人数
func (s *FileNameStatStore) loadTotalNames() (int, error) {
	path := filepath.Join(s.dataDir, "name_frequency.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("加载单字频率（语料元信息）失败: %w", err)
	}
	var freq struct {
		Meta struct {
			TotalNames int `json:"total_names"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(raw, &freq); err != nil {
		return 0, fmt.Errorf("解析 name_frequency.json 元信息失败: %w", err)
	}
	return freq.Meta.TotalNames, nil
}

func loadJSONFile[T any](dataDir, name string) ([]T, error) {
	raw, err := os.ReadFile(filepath.Join(dataDir, name))
	if err != nil {
		return nil, err
	}
	var items []T
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// trimLimit 负数或零 limit 视为不限，超出时截断到切片长度
func trimLimit(limit, max int) int {
	if limit <= 0 || limit > max {
		return max
	}
	return limit
}

func (s *FileNameStatStore) GetSurnameStats(limit int) ([]database.SurnameStat, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}
	return s.surnames[:trimLimit(limit, len(s.surnames))], nil
}

func (s *FileNameStatStore) GetSurnameStat(surname string) (*database.SurnameStat, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}
	for i := range s.surnames {
		if s.surnames[i].Surname == surname {
			stat := s.surnames[i]
			return &stat, nil
		}
	}
	return nil, nil
}

func (s *FileNameStatStore) GetGivenNameStats(surname string, limit int) ([]database.GivenNameStat, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}
	return filterBySurname(s.givenNames, surname, limit, func(g database.GivenNameStat) string {
		return g.Surname
	}), nil
}

func (s *FileNameStatStore) GetFullNameStats(surname string, limit int) ([]database.FullNameStat, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}
	return filterBySurname(s.fullNames, surname, limit, func(f database.FullNameStat) string {
		return f.Surname
	}), nil
}

func (s *FileNameStatStore) GetFullNameStat(fullName string) (*database.FullNameStat, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}
	for i := range s.fullNames {
		if s.fullNames[i].FullName == fullName {
			stat := s.fullNames[i]
			return &stat, nil
		}
	}
	return nil, nil
}

func (s *FileNameStatStore) GetNameGenderStats(name string) (*database.NameGenderStat, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}
	for i := range s.gender {
		if s.gender[i].Name == name {
			stat := s.gender[i]
			return &stat, nil
		}
	}
	return nil, nil
}

func (s *FileNameStatStore) GetTopFullNames(limit int) ([]database.FullNameStat, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}
	return s.fullNames[:trimLimit(limit, len(s.fullNames))], nil
}

func (s *FileNameStatStore) GetTotalNameCount() (int, error) {
	if err := s.ensureLoaded(); err != nil {
		return 0, err
	}
	return s.totalNames, nil
}

// filterBySurname 按姓氏过滤条目并截取前 limit 条
func filterBySurname[T any](items []T, surname string, limit int, getSurname func(T) string) []T {
	result := make([]T, 0, trimLimit(limit, len(items)))
	for _, it := range items {
		if getSurname(it) != surname {
			continue
		}
		result = append(result, it)
		if limit > 0 && len(result) == limit {
			break
		}
	}
	return result
}
