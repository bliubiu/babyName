package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/bliubiu/babyName/internal/domain/yijing"
	"github.com/bliubiu/babyName/internal/domain/zodiac"
	"github.com/bliubiu/babyName/internal/infrastructure/database"
)

type Store struct {
	db        *DB
	Hexagrams []yijing.Hexagram
	Zodiacs   []zodiac.Zodiac
}

// 确保Store实现了database.Store接口
var _ database.Store = (*Store)(nil)

func NewStore(db *DB) *Store {
	store := &Store{
		db:        db,
		Hexagrams: yijing.GetAllHexagrams(),
		Zodiacs:   zodiac.GetAllZodiacs(),
	}
	return store
}

func (s *Store) SaveHistory(record *database.HistoryRecord) string {
	if record.ID == "" {
		record.ID = uuid.New().String()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}

	query := `
	INSERT INTO history (id, surname, gender, birth_date, birth_time, birth_location, results, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	ON CONFLICT (id) DO UPDATE
	SET surname = $2, gender = $3, birth_date = $4, birth_time = $5, birth_location = $6, results = $7, created_at = $8
	`

	_, err := s.db.Exec(
		query,
		record.ID,
		record.Surname,
		record.Gender,
		record.BirthDate,
		record.BirthTime,
		record.BirthLocation,
		record.Results,
		record.CreatedAt,
	)
	if err != nil {
		fmt.Printf("Error saving history: %v\n", err)
	}

	return record.ID
}

func (s *Store) GetHistory() []*database.HistoryRecord {
	query := `SELECT id, surname, gender, birth_date, birth_time, birth_location, results, created_at FROM history ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		fmt.Printf("Error getting history: %v\n", err)
		return []*database.HistoryRecord{}
	}
	defer rows.Close()

	var records []*database.HistoryRecord
	for rows.Next() {
		var record database.HistoryRecord
		err := rows.Scan(
			&record.ID,
			&record.Surname,
			&record.Gender,
			&record.BirthDate,
			&record.BirthTime,
			&record.BirthLocation,
			&record.Results,
			&record.CreatedAt,
		)
		if err != nil {
			fmt.Printf("Error scanning history: %v\n", err)
			continue
		}
		records = append(records, &record)
	}

	return records
}

func (s *Store) GetHistoryByID(id string) *database.HistoryRecord {
	query := `SELECT id, surname, gender, birth_date, birth_time, birth_location, results, created_at FROM history WHERE id = $1`

	var record database.HistoryRecord
	err := s.db.QueryRow(query, id).Scan(
		&record.ID,
		&record.Surname,
		&record.Gender,
		&record.BirthDate,
		&record.BirthTime,
		&record.BirthLocation,
		&record.Results,
		&record.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		fmt.Printf("Error getting history by ID: %v\n", err)
		return nil
	}

	return &record
}

func (s *Store) DeleteHistory(id string) error {
	query := `DELETE FROM history WHERE id = $1`

	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete history: %w", err)
	}

	return nil
}

func (s *Store) BatchSaveHistory(records []*database.HistoryRecord) []string {
	ids := make([]string, len(records))
	for i, record := range records {
		ids[i] = s.SaveHistory(record)
	}
	return ids
}

func (s *Store) BatchDeleteHistory(ids []string) error {
	for _, id := range ids {
		if err := s.DeleteHistory(id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) SaveFavorite(record *database.FavoriteRecord) string {
	if record.ID == "" {
		record.ID = uuid.New().String()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}

	query := `
	INSERT INTO favorites (id, surname, given_name, pinyin, gender, score, source, notes, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	ON CONFLICT (id) DO UPDATE
	SET surname = $2, given_name = $3, pinyin = $4, gender = $5, score = $6, source = $7, notes = $8, created_at = $9
	`

	_, err := s.db.Exec(
		query,
		record.ID,
		record.Surname,
		record.GivenName,
		record.Pinyin,
		record.Gender,
		record.Score,
		record.Source,
		record.Notes,
		record.CreatedAt,
	)
	if err != nil {
		fmt.Printf("Error saving favorite: %v\n", err)
	}

	return record.ID
}

func (s *Store) GetFavorites() []*database.FavoriteRecord {
	query := `SELECT id, surname, given_name, pinyin, gender, score, source, notes, created_at FROM favorites ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		fmt.Printf("Error getting favorites: %v\n", err)
		return []*database.FavoriteRecord{}
	}
	defer rows.Close()

	var records []*database.FavoriteRecord
	for rows.Next() {
		var record database.FavoriteRecord
		err := rows.Scan(
			&record.ID,
			&record.Surname,
			&record.GivenName,
			&record.Pinyin,
			&record.Gender,
			&record.Score,
			&record.Source,
			&record.Notes,
			&record.CreatedAt,
		)
		if err != nil {
			fmt.Printf("Error scanning favorite: %v\n", err)
			continue
		}
		records = append(records, &record)
	}

	return records
}

func (s *Store) DeleteFavorite(id string) error {
	query := `DELETE FROM favorites WHERE id = $1`

	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete favorite: %w", err)
	}

	return nil
}

func (s *Store) BatchSaveFavorite(records []*database.FavoriteRecord) []string {
	ids := make([]string, len(records))
	for i, record := range records {
		ids[i] = s.SaveFavorite(record)
	}
	return ids
}

func (s *Store) BatchDeleteFavorite(ids []string) error {
	for _, id := range ids {
		if err := s.DeleteFavorite(id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) GetFavoriteByName(surname, givenName string) *database.FavoriteRecord {
	query := `SELECT id, surname, given_name, pinyin, gender, score, source, notes, created_at FROM favorites WHERE surname = $1 AND given_name = $2`

	var record database.FavoriteRecord
	err := s.db.QueryRow(query, surname, givenName).Scan(
		&record.ID,
		&record.Surname,
		&record.GivenName,
		&record.Pinyin,
		&record.Gender,
		&record.Score,
		&record.Source,
		&record.Notes,
		&record.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		fmt.Printf("Error getting favorite by name: %v\n", err)
		return nil
	}

	return &record
}

func (s *Store) GetHexagramByNumber(id int) *yijing.Hexagram {
	for i := range s.Hexagrams {
		if s.Hexagrams[i].Number == id {
			return &s.Hexagrams[i]
		}
	}
	return nil
}

func (s *Store) GetHexagrams() []yijing.Hexagram {
	result := make([]yijing.Hexagram, len(s.Hexagrams))
	copy(result, s.Hexagrams)
	return result
}

func (s *Store) GetZodiacByName(name string) *zodiac.Zodiac {
	for i := range s.Zodiacs {
		if s.Zodiacs[i].Name == name {
			return &s.Zodiacs[i]
		}
	}
	return nil
}

func (s *Store) GetZodiacs() []zodiac.Zodiac {
	result := make([]zodiac.Zodiac, len(s.Zodiacs))
	copy(result, s.Zodiacs)
	return result
}
