package storage

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

type Storage struct {
	db *sql.DB
}

func NewStorage() (*Storage, error) {
	db, err := sql.Open("sqlite3", "./ombibot.db")
	if err != nil {
		return nil, err
	}

	db.Exec("CREATE TABLE IF NOT EXISTS search_results (uuid TEXT PRIMARY KEY, results TEXT)")

	return &Storage{db: db}, nil
}

func (storage *Storage) Close() {
	storage.db.Close()
}

func (storage *Storage) SaveSearchResults(uuid string, results string) error {
	_, err := storage.db.Exec("INSERT INTO search_results (uuid, results) VALUES (?, ?)", uuid, results)
	if err != nil {
		return fmt.Errorf("error saving search results: %s", err)
	} else {
		return nil
	}
}

func (s *Storage) GetSearchResults(uuid string) (string, error) {
	rows, err := s.db.Query("SELECT results FROM search_results WHERE uuid = ?", uuid)
	if err != nil {
		return "", fmt.Errorf("error getting search results: %s", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return "", fmt.Errorf("no results found for uuid %s", uuid)
	}

	var results_json string
	err = rows.Scan(&results_json)
	if err != nil {
		return "", fmt.Errorf("error scanning results: %s", err)
	}

	return results_json, nil
}
