package sqlite

import (
	_ "modernc.org/sqlite"
	"name/internal/infrastructure/database"
)

func (s *Store) GetSurnameStats(limit int) ([]database.SurnameStat, error) {
	return nil, errNameStatsUnavailable
}
func (s *Store) GetSurnameStat(surname string) (*database.SurnameStat, error) {
	return nil, errNameStatsUnavailable
}
func (s *Store) GetGivenNameStats(surname string, limit int) ([]database.GivenNameStat, error) {
	return nil, errNameStatsUnavailable
}
func (s *Store) GetFullNameStats(surname string, limit int) ([]database.FullNameStat, error) {
	return nil, errNameStatsUnavailable
}
func (s *Store) GetFullNameStat(fullName string) (*database.FullNameStat, error) {
	return nil, errNameStatsUnavailable
}
func (s *Store) GetNameGenderStats(name string) (*database.NameGenderStat, error) {
	return nil, errNameStatsUnavailable
}
func (s *Store) GetTopFullNames(limit int) ([]database.FullNameStat, error) {
	return nil, errNameStatsUnavailable
}
func (s *Store) GetTotalNameCount() (int, error) {
	return 0, errNameStatsUnavailable
}
