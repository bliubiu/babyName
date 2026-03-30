package postgres

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func NewDB(host, port, user, password, dbname string) (*DB, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Connected to PostgreSQL database")

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return &DB{db}, nil
}

func createTables(db *sql.DB) error {
	// Create history table
	historyTable := `
	CREATE TABLE IF NOT EXISTS history (
		id VARCHAR(36) PRIMARY KEY,
		surname VARCHAR(50) NOT NULL,
		gender VARCHAR(10) NOT NULL,
		birth_date VARCHAR(20) NOT NULL,
		birth_time VARCHAR(20) NOT NULL,
		birth_location VARCHAR(100),
		results TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL
	);
	`

	// Create favorites table
	favoritesTable := `
	CREATE TABLE IF NOT EXISTS favorites (
		id VARCHAR(36) PRIMARY KEY,
		surname VARCHAR(50) NOT NULL,
		given_name VARCHAR(50) NOT NULL,
		pinyin VARCHAR(100) NOT NULL,
		gender VARCHAR(10) NOT NULL,
		score INTEGER NOT NULL,
		source VARCHAR(100),
		notes TEXT,
		created_at TIMESTAMP NOT NULL
	);
	`

	// Create index on favorites for faster lookup
	favoritesIndex := `
	CREATE INDEX IF NOT EXISTS idx_favorites_name ON favorites(surname, given_name);
	`

	// Create index on history for faster lookup by date
	historyIndex := `
	CREATE INDEX IF NOT EXISTS idx_history_created_at ON history(created_at);
	`

	queries := []string{historyTable, favoritesTable, favoritesIndex, historyIndex}

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	log.Println("Created database tables and indexes")
	return nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}
