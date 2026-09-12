package namestatistics

import (
	"errors"
	"testing"

	"name/internal/infrastructure/database"
)

// fakeNameStatStore NameStatStore 接口的假实现，用于验证服务层透传行为
type fakeNameStatStore struct {
	surnameStats   []database.SurnameStat
	surnameStat    *database.SurnameStat
	givenNameStats []database.GivenNameStat
	fullNameStats  []database.FullNameStat
	fullNameStat   *database.FullNameStat
	genderStats    *database.NameGenderStat
	totalCount     int
	err            error
}

var _ database.NameStatStore = (*fakeNameStatStore)(nil)

func (f *fakeNameStatStore) GetSurnameStats(limit int) ([]database.SurnameStat, error) {
	return f.surnameStats, f.err
}
func (f *fakeNameStatStore) GetSurnameStat(surname string) (*database.SurnameStat, error) {
	return f.surnameStat, f.err
}
func (f *fakeNameStatStore) GetGivenNameStats(surname string, limit int) ([]database.GivenNameStat, error) {
	return f.givenNameStats, f.err
}
func (f *fakeNameStatStore) GetFullNameStats(surname string, limit int) ([]database.FullNameStat, error) {
	return f.fullNameStats, f.err
}
func (f *fakeNameStatStore) GetFullNameStat(fullName string) (*database.FullNameStat, error) {
	return f.fullNameStat, f.err
}
func (f *fakeNameStatStore) GetNameGenderStats(name string) (*database.NameGenderStat, error) {
	return f.genderStats, f.err
}
func (f *fakeNameStatStore) GetTopFullNames(limit int) ([]database.FullNameStat, error) {
	return f.fullNameStats, f.err
}
func (f *fakeNameStatStore) GetTotalNameCount() (int, error) {
	return f.totalCount, f.err
}

func newFakeStore() *fakeNameStatStore {
	return &fakeNameStatStore{
		surnameStats: []database.SurnameStat{
			{Surname: "王", Count: 100, Rank: 1},
			{Surname: "李", Count: 90, Rank: 2},
		},
		surnameStat: &database.SurnameStat{Surname: "王", Count: 100, Rank: 1},
		givenNameStats: []database.GivenNameStat{
			{Surname: "王", GivenName: "伟", Count: 10, Rank: 1},
		},
		fullNameStats: []database.FullNameStat{
			{FullName: "王伟", Surname: "王", GivenName: "伟", Count: 5, Rank: 1},
		},
		fullNameStat: &database.FullNameStat{FullName: "王伟", Count: 5, Rank: 1},
		genderStats: &database.NameGenderStat{
			Name: "伟", MaleCount: 4, FemaleCount: 0, TotalCount: 5,
			MaleRatio: 0.8, FemaleRatio: 0.0,
		},
		totalCount: 12345,
	}
}

// TestNameStatisticsService 验证领域服务对 NameStatStore 的透传（含错误传播）
func TestNameStatisticsService(t *testing.T) {
	store := newFakeStore()
	svc := NewNameStatisticsService(store)

	t.Run("姓氏排行透传", func(t *testing.T) {
		stats, err := svc.GetSurnameStats(10)
		if err != nil {
			t.Fatalf("GetSurnameStats 失败: %v", err)
		}
		if len(stats) != 2 || stats[0].Surname != "王" {
			t.Errorf("GetSurnameStats 未透传: %+v", stats)
		}
	})

	t.Run("单个姓氏透传", func(t *testing.T) {
		stat, err := svc.GetSurnameStat("王")
		if err != nil {
			t.Fatalf("GetSurnameStat 失败: %v", err)
		}
		if stat == nil || stat.Count != 100 {
			t.Errorf("GetSurnameStat 未透传: %+v", stat)
		}
	})

	t.Run("名字排行透传", func(t *testing.T) {
		stats, err := svc.GetGivenNameStats("王", 10)
		if err != nil {
			t.Fatalf("GetGivenNameStats 失败: %v", err)
		}
		if len(stats) != 1 || stats[0].GivenName != "伟" {
			t.Errorf("GetGivenNameStats 未透传: %+v", stats)
		}
	})

	t.Run("全名排行透传", func(t *testing.T) {
		stats, err := svc.GetFullNameStats("王", 10)
		if err != nil {
			t.Fatalf("GetFullNameStats 失败: %v", err)
		}
		if len(stats) != 1 || stats[0].FullName != "王伟" {
			t.Errorf("GetFullNameStats 未透传: %+v", stats)
		}
	})

	t.Run("单全名透传", func(t *testing.T) {
		stat, err := svc.GetFullNameStat("王伟")
		if err != nil {
			t.Fatalf("GetFullNameStat 失败: %v", err)
		}
		if stat == nil || stat.FullName != "王伟" {
			t.Errorf("GetFullNameStat 未透传: %+v", stat)
		}
	})

	t.Run("性别分布透传", func(t *testing.T) {
		stat, err := svc.GetNameGenderStats("伟")
		if err != nil {
			t.Fatalf("GetNameGenderStats 失败: %v", err)
		}
		if stat == nil || stat.TotalCount != 5 {
			t.Errorf("GetNameGenderStats 未透传: %+v", stat)
		}
	})

	t.Run("热门全名透传", func(t *testing.T) {
		stats, err := svc.GetTopFullNames(10)
		if err != nil {
			t.Fatalf("GetTopFullNames 失败: %v", err)
		}
		if len(stats) != 1 {
			t.Errorf("GetTopFullNames 未透传: %+v", stats)
		}
	})

	t.Run("总人名数透传", func(t *testing.T) {
		count, err := svc.GetTotalNameCount()
		if err != nil {
			t.Fatalf("GetTotalNameCount 失败: %v", err)
		}
		if count != 12345 {
			t.Errorf("GetTotalNameCount 未透传: %d", count)
		}
	})

	t.Run("错误传播", func(t *testing.T) {
		errStore := &fakeNameStatStore{err: errors.New("数据源不可用")}
		badSvc := NewNameStatisticsService(errStore)
		_, err := badSvc.GetSurnameStats(10)
		if err == nil {
			t.Fatal("GetSurnameStats 应传播 store 错误，实际 nil")
		}
		_, err = badSvc.GetTotalNameCount()
		if err == nil {
			t.Fatal("GetTotalNameCount 应传播 store 错误，实际 nil")
		}
	})
}

// TestNewNameStatisticsService 验证构造函数的接口返回值
func TestNewNameStatisticsService(t *testing.T) {
	svc := NewNameStatisticsService(newFakeStore())
	if svc == nil {
		t.Fatal("构造函数返回 nil")
	}
}
