package main

type UserRepository interface {
	InsertUser(user User) error
	close()
}

type LinkRepository interface {
	InsertLink(link Link) error
	GetLinkById(id uint64) (*Link, error)
	UpdateLink(link Link) error
	close()
}

type VoteRepository interface {
	MarkVote(link Link, user User) error
	close()
}

type TestRepository interface {
	NewTest(message string) error
	close()
}
