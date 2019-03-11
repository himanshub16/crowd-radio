package main

type UserRepository interface {
	CreateOrUpdateUser(user User) error
	GetUserByID(userID string) *User
	close()
}

type LinkRepository interface {
	InsertLink(link Link) int64
	GetLinkByID(id int64) (*Link, error)
	GetAllLinks(limit int64) []Link
	GetLinksByUser(userID string) []Link
	UpdateLink(link Link) error
	GetVotesForUser(linkIDs []int64, userID string) map[int64]int64
	TotalVotesForLink(linkID int64) int64
	close()
}

type VoteRepository interface {
	MarkVote(linkID int64, userID string, score int64) error
	close()
}

type TestRepository interface {
	NewTest(message string) error
	close()
}
