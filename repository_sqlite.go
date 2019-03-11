package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
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
	  insert into links (url, video_id, title, channel_name, duration,
						submitted_by, dedicated_to, is_expired, created_at)
	  values (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(link.URL, link.VideoID, link.Title, link.ChannelName, link.Duration,
		link.SubmittedBy, link.DedicatedTo, link.IsExpired, link.CreatedAt)
	if err != nil {
		log.Fatal(err)
	}
	linkID, _ := res.LastInsertId()

	return linkID
}

func (r *SQLiteRepository) GetLinkByID(id int64) (*Link, error) {
	stmt, err := r.db.Prepare(`
	  select l.link_id, l.video_id, l.url, l.title, l.channel_name, l.duration,
		l.submitted_by, l.dedicated_to, l.is_expired, l.created_at,
		(select coalesce(sum(score), 0) from votes as v where v.link_id = l.link_id)
	  from links as l
      where l.link_id=?
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	l := Link{LinkID: id}
	err = stmt.QueryRow(l.LinkID).Scan(&l.LinkID, &l.VideoID, &l.URL, &l.Title, &l.ChannelName,
		&l.Duration, &l.SubmittedBy, &l.DedicatedTo, &l.IsExpired, &l.CreatedAt,
		&l.TotalVotes)

	return &l, err
}

func (r *SQLiteRepository) GetLinksByUser(userID string) []Link {
	stmt, err := r.db.Prepare(`
	  select l.link_id, l.video_id, l.url, l.title, l.channel_name, l.duration,
		l.submitted_by, l.dedicated_to, l.is_expired, l.created_at,
		(select coalesce(sum(score), 0) from votes as v1 where v1.link_id = l.link_id),
		(select coalesce(sum(score), 0) from votes as v2 where v2.link_id = l.link_id and v2.user_id = l.submitted_by)
	  from links as l
      where l.is_expired=false and l.submitted_by=?
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(userID)
	if err != nil {
		log.Fatal(err)
	}

	links := make([]Link, 0)
	for rows.Next() {
		l := Link{}
		err = rows.Scan(&l.LinkID, &l.VideoID, &l.URL, &l.Title, &l.ChannelName,
			&l.Duration, &l.SubmittedBy, &l.DedicatedTo, &l.IsExpired, &l.CreatedAt,
			&l.TotalVotes, &l.MyVote)
		if err != nil {
			log.Fatal(err)
		}

		links = append(links, l)
	}
	return links
}

func (r *SQLiteRepository) UpdateLink(link Link) error {
	stmt, err := r.db.Prepare(`
	  update links
	  set url=?, title=?, channel_name=?, duration=?,
		submitted_by=?, dedicated_to=?, is_expired=?, created_at=?
	  where link_id=?
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(link.URL, link.Title, link.ChannelName, link.Duration,
		link.SubmittedBy, link.DedicatedTo, link.IsExpired, link.CreatedAt,
		link.LinkID)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func (r *SQLiteRepository) GetAllLinks(limit int64) []Link {
	stmt, err := r.db.Prepare(`
	  select l.link_id, l.video_id, l.url, l.title, l.channel_name, l.duration,
		l.submitted_by, l.dedicated_to, l.is_expired, l.created_at,
		(select coalesce(sum(score), 0) from votes as v where v.link_id = l.link_id)
	  from links as l
      where l.is_expired=false
      limit ?
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(limit)
	if err != nil {
		log.Println("links by user failed here")
		log.Fatal(err)
	}

	links := make([]Link, 0)
	for rows.Next() {
		l := Link{}
		err = rows.Scan(&l.LinkID, &l.VideoID, &l.URL, &l.Title, &l.ChannelName,
			&l.Duration, &l.SubmittedBy, &l.DedicatedTo, &l.IsExpired, &l.CreatedAt,
			&l.TotalVotes)
		if err != nil {
			log.Fatal(err)
		}

		links = append(links, l)
	}
	return links
}

func (r *SQLiteRepository) GetVotesForUser(linkIds []int64, userID string) map[int64]int64 {
	var query string
	if len(linkIds) > 0 {
		query = "select link_id, score from votes where user_id=? and link_id in (?" +
			strings.Repeat(",?", len(linkIds)-1) +
			")"
	} else {
		return make(map[int64]int64)
	}
	stmt, err := r.db.Prepare(query)
	if err != nil {
		log.Fatal(err)
	}

	args := make([]interface{}, 0)
	args = append(args, userID)
	for _, lid := range linkIds {
		var tmp interface{}
		tmp = lid
		args = append(args, tmp)
	}

	rows, err := stmt.Query(args...)
	if err != nil {
		log.Fatal(err)
	}

	result := make(map[int64]int64)
	for rows.Next() {
		var linkid, score int64
		rows.Scan(&linkid, &score)
		result[linkid] = score
	}
	return result
}

func (r *SQLiteRepository) MarkVote(linkID int64, userID string, score int64) error {
	stmt, err := r.db.Prepare(`
	  replace into votes(link_id, user_id, score)
	  values (?, ?, ?)
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(linkID, userID, score)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func (r *SQLiteRepository) TotalVoteForLinks(linkIDs []int64) map[int64]int64 {
	var query string
	if len(linkIDs) > 0 {
		query = "select link_id, sum(score) from votes where link_id in (?" +
			strings.Repeat(",?", len(linkIDs)-1) +
			") group by link_id"
	} else {
		return make(map[int64]int64)
	}
	stmt, err := r.db.Prepare(query)
	if err != nil {
		log.Fatal(err)
	}

	args := make([]interface{}, 0)
	for _, lid := range linkIDs {
		var tmp interface{}
		tmp = lid
		args = append(args, tmp)
	}

	rows, err := stmt.Query(args...)
	if err != nil {
		log.Fatal(err)
	}

	result := make(map[int64]int64)
	for rows.Next() {
		var linkid, score int64
		rows.Scan(&linkid, &score)
		result[linkid] = score
	}
	return result
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
		video_id text not null,
		title text,
		channel_name text,
		duration int,
		submitted_by int,
		dedicated_to text,
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
