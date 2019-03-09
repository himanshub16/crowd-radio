package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

type SQLiteRepository struct {
	db *sql.DB
}

func (r *SQLiteRepository) CreateOrUpdateUser(user User) error {
	stmt, err := r.db.Prepare(`
	  replace into users (user_id, firstname, lastname, email)
	  values (?, ?, ?, ?)
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(user.UserID, user.FirstName, user.LastName, user.Email)
	if err != nil {
		log.Fatal(err)
	}

	return err
}

func (r *SQLiteRepository) GetUserByID(userID string) *User {
	stmt, err := r.db.Prepare(`
	  select user_id, firstname, lastname, email
	  from users where user_id= ?
	  `)
	if err != nil {
		log.Fatal("failed to prepare stmt ", err)
	}
	defer stmt.Close()

	user := &User{}
	err = stmt.QueryRow(userID).Scan(&user.UserID, &user.FirstName, &user.LastName, &user.Email)
	defer stmt.Close()
	if err != nil {
		log.Fatal("failed to find user", err)
	}

	return user
}

func (r *SQLiteRepository) InsertLink(link Link) int64 {
	stmt, err := r.db.Prepare(`
	  insert into links (url, title, channel_name, duration,
						submitted_by, dedicated_to, total_votes, is_expired, created_at)
	  values (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(link.URL, link.Title, link.ChannelName, link.Duration, link.SubmittedBy, link.DedicatedTo, link.TotalVotes, link.IsExpired, link.CreatedAt)
	if err != nil {
		log.Fatal(err)
	}
	linkID, _ := res.LastInsertId()

	return linkID
}

func (r *SQLiteRepository) GetLinkById(id uint64) (*Link, error) {
	return nil, nil
}

func (r *SQLiteRepository) UpdateLink(link Link) error {
	return nil
}

func (r *SQLiteRepository) GetAllLinks() []Link {
	links := make([]Link, 10)
	for i, _ := range links {
		links[i].Duration = 10
		links[i].LinkID = int64(i + 1)
	}
	return links
}

func (r *SQLiteRepository) MarkVote(linkID int64, userID string, score int64) {
	stmt, err := r.db.Prepare(`
	  replace into votes(link_id, user_id, score)
	  values (?, ?, ?)
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	fmt.Println(linkID, userID, score)
	if _, err := stmt.Exec(linkID, userID, score); err != nil {
		log.Fatal(err)
	}
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
	if err != nil {
		return nil
	}

	// make sure the required tables exist
	// if not then create them
	testTable := `
	  create table if not exists test (
		message text
	  )`
	usersTable := `
	  create table if not exists users (
		user_id int primary key,
		firstname text,
		lastname text,
		email text
	  )`

	linksTable := `
		create table if not exists links (
		link_id integer primary key autoincrement,
		url text not null,
		title text,
		channel_name text,
		duration int,
		submitted_by int,
		dedicated_to text,
		total_votes int,
		is_expired bool,
		created_at int
	  )`
	votesTable := `
		create table if not exists votes (
		link_id integer not null,
		user_id integer not null,
		score integer not null,
		constraint unq UNIQUE(link_id, user_id)
	  )`

	tables := []string{testTable, usersTable, linksTable, votesTable}
	var stmt *sql.Stmt

	for _, t := range tables {
		if stmt, err = db.Prepare(t); err != nil {
			log.Fatal("Failed to prepare stmt", err)
		}
		defer stmt.Close()
		if _, err = stmt.Exec(); err != nil {
			log.Fatal("failed to exec stmt", err)
		}
	}
	// check for possible errors and traps

	return &SQLiteRepository{db: db}
}
