package main

import (
	"log"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type PostgresRepository struct {
	db *sqlx.DB
}

func (r *PostgresRepository) CreateOrUpdateUser(user User) error {
	query := `
      insert into users (user_id, firstname, lastname, email)
      values ($1, $2, $3, $4)
      on conflict(user_id) do update
         set firstname = excluded.firstname,
             lastname = excluded.lastname,
             email = excluded.email;`

	_, err := r.db.Exec(query, user.UserID, user.FirstName, user.LastName, user.Email)
	if err != nil {
		log.Fatal(err)
	}
	return err
}

func (r *PostgresRepository) GetUserByID(userID string) *User {
	query := `
	  select user_id, firstname, lastname, email
	  from users where user_id=$1;
	  `

	user := &User{}
	err := r.db.QueryRow(query, userID).Scan(&user.UserID, &user.FirstName, &user.LastName, &user.Email)
	if err != nil {
		log.Fatal("failed to find user", err)
	}

	return user
}

func (r *PostgresRepository) InsertLink(link Link) int64 {
	query := `
	  insert into links (url, video_id, title, channel_name, duration,
						submitted_by, dedicated_to, is_expired, created_at)
	  values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
      returning link_id;
    `

	var linkId int64
	err := r.db.QueryRow(query, link.URL, link.VideoID, link.Title, link.ChannelName,
		link.Duration, link.SubmittedBy, link.DedicatedTo, link.IsExpired, link.CreatedAt,
	).Scan(&linkId)

	if err != nil {
		log.Fatal(err)
	}
	return linkId
}

func (r *PostgresRepository) GetLinkByID(id int64) (*Link, error) {
	query := `
	  select l.link_id, l.video_id, l.url, l.title, l.channel_name, l.duration,
		l.submitted_by, l.dedicated_to, l.is_expired, l.created_at,
		(select coalesce(sum(score), 0) from votes as v where v.link_id = l.link_id)
	  from links as l
	  where l.link_id=$1;`

	l := Link{LinkID: id}
	err := r.db.QueryRow(query, l.LinkID).Scan(&l.LinkID, &l.VideoID, &l.URL, &l.Title, &l.ChannelName,
		&l.Duration, &l.SubmittedBy, &l.DedicatedTo, &l.IsExpired, &l.CreatedAt, &l.TotalVotes)

	return &l, err
}

func (r *PostgresRepository) GetLinksByUser(userID string) []Link {
	query := `
	  select l.link_id, l.video_id, l.url, l.title, l.channel_name, l.duration,
		l.submitted_by, l.dedicated_to, l.is_expired, l.created_at,
		(select coalesce(sum(score), 0) from votes as v1 where v1.link_id = l.link_id),
		(select coalesce(sum(score), 0) from votes as v2 where v2.link_id = l.link_id and v2.user_id = l.submitted_by)
	  from links as l
      where l.is_expired=false and l.submitted_by=$1;
    `

	rows, err := r.db.Query(query, userID)
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

func (r *PostgresRepository) UpdateLink(link Link) error {
	query := `
	  update links
	  set url=$1, title=$2, channel_name=$3, duration=$4,
		submitted_by=$5, dedicated_to=$6, is_expired=$7, created_at=$8
	  where link_id=$9;`

	_, err := r.db.Exec(query, link.URL, link.Title, link.ChannelName, link.Duration,
		link.SubmittedBy, link.DedicatedTo, link.IsExpired, link.CreatedAt, link.LinkID)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func (r *PostgresRepository) GetAllLinks(limit int64) []Link {
	query := `
	  select l.link_id, l.video_id, l.url, l.title, l.channel_name, l.duration,
		l.submitted_by, l.dedicated_to, l.is_expired, l.created_at,
		(select coalesce(sum(score), 0) from votes as v where v.link_id = l.link_id)
	  from links as l
      where l.is_expired=false
      limit $1;`

	rows, err := r.db.Query(query, limit)
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

func (r *PostgresRepository) GetVotesForUser(linkIds []int64, userID string) map[int64]int64 {
	if len(linkIds) == 0 {
		return make(map[int64]int64)
	}

	query, args, err := sqlx.In(
		`select link_id, score from votes where user_id=$1 and link_id IN (?);`,
		userID, linkIds)
	query = r.db.Rebind(query)
	rows, err := r.db.Query(query, args...)
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

func (r *PostgresRepository) MarkVote(linkID int64, userID string, score int64) error {
	query := `
	  insert into votes(link_id, user_id, score)
	  values ($1, $2, $3)
      on conflict(link_id, user_id) do update
         set user_id=excluded.user_id,
             link_id=excluded.link_id,
             score=excluded.score;
	`
	_, err := r.db.Exec(query, linkID, userID, score)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func (r *PostgresRepository) TotalVoteForLinks(linkIDs []int64) map[int64]int64 {
	var query string
	if len(linkIDs) > 0 {
		query = "select link_id, sum(score) from votes where link_id in (?" +
			strings.Repeat(",?", len(linkIDs)-1) +
			") group by link_id;"
	} else {
		return make(map[int64]int64)
	}

	query = r.db.Rebind(query)

	args := make([]interface{}, 0)
	for _, lid := range linkIDs {
		var tmp interface{}
		tmp = lid
		args = append(args, tmp)
	}

	rows, err := r.db.Query(query, args...)
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

func (r *PostgresRepository) NewTest(message string) error {
	query := `INSERT INTO test (message) values ($1)`
	res, err := r.db.Exec(query, message)
	naff, _ := res.RowsAffected()
	log.Println("new test ", naff, " rows affected")
	return err
}

func (r *PostgresRepository) close() {
	r.db.Close()
}

// func NewPostgresRepository(host, port, user, pass, dbname string) *PostgresRepository {
func NewPostgresRepository(dbUrl string) *PostgresRepository {

	// psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
	// 	host, port, user, pass, dbname)

	// db, err := sqlx.Open("postgres", psqlInfo)
	db, err := sqlx.Open("postgres", dbUrl)
	if err != nil {
		return nil
	}
	log.Println("connected to db. creating new tables")

	// make sure the required tables exist
	// if not then create them
	testTable := `
	  create table if not exists test (
		message text
	  );`
	usersTable := `
	  create table if not exists users (
		user_id text primary key,
		firstname text,
		lastname text,
		email text
	  );`

	linksTable := `
		create table if not exists links (
		link_id serial primary key,
		url text not null,
		video_id text not null,
		title text,
		channel_name text,
		duration int,
		submitted_by text,
		dedicated_to text,
		is_expired bool,
		created_at int
	  );`
	votesTable := `
		create table if not exists votes (
		link_id integer not null,
		user_id text not null,
		score integer not null,
		constraint unq UNIQUE(link_id, user_id)
	  );`

	tables := []string{testTable, usersTable, linksTable, votesTable}

	for _, t := range tables {
		if _, err = db.Exec(t); err != nil {
			log.Fatal("failed to exec stmt ", err)
		}
	}
	// check for possible errors and traps
	log.Println("Connected to database", db)
	return &PostgresRepository{db: db}
}
