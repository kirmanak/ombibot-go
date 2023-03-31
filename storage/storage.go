package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type Storage interface {
	Close()
	SaveSearchResults(uuid string, results string) error
	GetSearchResults(uuid string) (string, error)
}

type Sqlite struct {
	db *sql.DB
}

func NewStorage() (*Sqlite, error) {
	db, err := sql.Open("sqlite3", "./ombibot.db")
	if err != nil {
		return nil, err
	}

	db.Exec("CREATE TABLE IF NOT EXISTS search_results (uuid TEXT PRIMARY KEY, results TEXT)")

	return &Sqlite{db: db}, nil
}

func (storage *Sqlite) Close() {
	storage.db.Close()
}

func (storage *Sqlite) SaveSearchResults(uuid string, results string) error {
	_, err := storage.db.Exec("INSERT INTO search_results (uuid, results) VALUES (?, ?)", uuid, results)
	if err != nil {
		return fmt.Errorf("error saving search results: %w", err)
	} else {
		return nil
	}
}

func (storage *Sqlite) GetSearchResults(uuid string) (string, error) {
	rows, err := storage.db.Query("SELECT results FROM search_results WHERE uuid = ?", uuid)
	if err != nil {
		return "", fmt.Errorf("error getting search results: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return "", fmt.Errorf("no results found for uuid " + uuid)
	}

	var results_json string
	err = rows.Scan(&results_json)
	if err != nil {
		return "", fmt.Errorf("error scanning results: %w", err)
	}

	return results_json, nil
}
