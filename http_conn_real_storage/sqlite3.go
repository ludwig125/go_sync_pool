package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type sqliteRepository struct {
	db *sql.DB
}

// interfaceを実装しているか保証する
// See: http://golang.org/doc/faq#guarantee_satisfies_interface
var _ PersonRepository = (*sqliteRepository)(nil)

func NewSQLitePersonRepository(dbName string) (PersonRepository, error) {
	// 対象のDBがなくても新規に作ってしまうようなので、DBファイルの存在確認する
	if !exists(dbName) {
		return nil, fmt.Errorf("no such db file: %s", dbName)
	}

	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		return nil, fmt.Errorf("failed to connection db: %v", err)
	}
	log.Printf("connected %s successfully", dbName)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %v", err)
	}
	log.Printf("ping %s successfully", dbName)
	return &sqliteRepository{db: db}, nil
}

func exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func (r *sqliteRepository) FindByID(id int) ([]Person, error) {
	rows, err := r.db.Query("SELECT * FROM person WHERE id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v", err)
	}
	return scanPersons(rows)
}

func scanPersons(rows *sql.Rows) ([]Person, error) {
	var people []Person
	defer rows.Close()
	for rows.Next() {
		var p Person
		err := rows.Scan(&p.ID, &p.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to scan: %v", err)
		}
		people = append(people, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row error: %v", err)
	}
	return people, nil
}
