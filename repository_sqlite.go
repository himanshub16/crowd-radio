package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

type SQLiteRepository struct {
	db *sql.DB
}

func (r *SQLiteRepository) InsertUser(user User) error {
	// perform sql query here
	return nil
}

func (r *SQLiteRepository) InsertLink(link Link) error {
	return nil
}

func (r *SQLiteRepository) GetLinkById(id uint64) (*Link, error) {
	return nil, nil
}

func (r *SQLiteRepository) UpdateLink(link Link) error {
	return nil
}

func (r *SQLiteRepository) MarkVote(link Link, user User) error {
	return nil
}

func (r *SQLiteRepository) NewTest(message string) error {
	fmt.Println("performing query")
	stmt, err := r.db.Prepare("INSERT INTO test(message) values(?)")
	res, err := stmt.Exec(message)
	fmt.Println(res.LastInsertId())
	return err
}

func (r *SQLiteRepository) close() {
	r.db.Close()
}

func NewSQLiteRepository(filePath string) *SQLiteRepository {
	fmt.Println("got", filePath, "but using", "db.sqlite3")
	db, err := sql.Open("sqlite3", "db.sqlite3")

	// make sure the required tables exist
	// if not then create them
	createTestTableQuery := `CREATE TABLE IF NOT EXISTS test (message VARCHAR(64))`
	stmt, _ := db.Prepare(createTestTableQuery)
	res, _ := stmt.Exec()
	fmt.Println(res)
	// check for possible errors and traps

	if err != nil {
		return nil
	}
	return &SQLiteRepository{db: db}
}
