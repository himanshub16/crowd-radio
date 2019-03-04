// this file defines the data structures to be used throught
package main

import (
	"time"
)

type Link struct {
	LinkID     uint64     `json:"link_id"`
	URL        string     `json:"url"`
	TotalVotes int64      `json:"total_votes"`
	IsExpired  bool       `json:"is_expired"`
	CreatedAt  *time.Time `json:"created_at"`
}

type Vote struct {
	UserID string `json:"user_id"`
	LinkID uint64 `json:"link_id"`
	Score  int    `json:"score"`
}

type User struct {
	UserID uint64 `json:"user_id"`
	Email  string `json:"email"`
}
