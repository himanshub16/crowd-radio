// this file defines the data structures to be used throught
package main

type Link struct {
	LinkID      int64  `json:"link_id"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	ChannelName string `json:"channel_name"`
	Duration    int64  `json:"duration"`
	SubmittedBy string `json:"submitted_by"`
	DedicatedTo string `json:"dedicated_to"`
	TotalVotes  int64  `json:"total_votes"`
	IsExpired   bool   `json:"is_expired"`
	CreatedAt   int64  `json:"created_at"`
}

type Vote struct {
	UserID string `json:"user_id"`
	LinkID int64  `json:"link_id"`
	Score  int    `json:"score"`
}

type User struct {
	UserID    string `json:"user_id"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Email     string `json:"email"`
}
