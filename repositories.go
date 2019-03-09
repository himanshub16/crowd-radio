package main

type UserRepository interface {
	CreateOrUpdateUser(user User) error
	GetUserByID(userID string) *User
	close()
}

type LinkRepository interface {
	InsertLink(link Link) int64
	GetLinkById(id uint64) (*Link, error)
	GetAllLinks() []Link
	UpdateLink(link Link) error
	close()
}

type VoteRepository interface {
	MarkVote(linkID int64, userID string, score int64)
	close()
}

type TestRepository interface {
	NewTest(message string) error
	close()
}
