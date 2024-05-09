package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

func DBConnection() *sql.DB {

	db, err := sql.Open("sqlite3", "./data/quotes.sqlite")
	if err != nil {
		log.Fatal("Error opening database connection:", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS quote (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		quote TEXT,
		name TEXT,
		author TEXT
	);`)
	if err != nil {
		log.Fatal("Error creating table:", err)
	}

	return db
}
