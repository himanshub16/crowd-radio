package main

import (
	"fmt"
	"time"
)

type Service interface {
	CreateOrUpdateUser(u User) error
	SubmitLink(url, userid, dedicatedTo string) (*Link, error)
	Vote(linkID int64, userID string, score int64)
	Test(message string)
	GetAllLinks() []Link
	GetLinkByID(linkID int64) (*Link, error)
	GetLinksByUser(userID string) []Link
	GetVotesForUser(links []Link, userID string) map[int64]int64
	close()
}

type ServiceImpl struct {
	linkRepo LinkRepository
	userRepo UserRepository
	voteRepo VoteRepository
	testRepo TestRepository
}

func (s *ServiceImpl) GetLinkByID(linkID int64) (*Link, error) {
	return s.linkRepo.GetLinkByID(linkID)
}

func (s *ServiceImpl) CreateOrUpdateUser(u User) error {
	return s.userRepo.CreateOrUpdateUser(u)
}

func (s *ServiceImpl) SubmitLink(url, userid, dedicatedTo string) (*Link, error) {
	// required checks here
	link := Link{
		URL:         url,
		SubmittedBy: userid,
		DedicatedTo: dedicatedTo,
		IsExpired:   false,
		CreatedAt:   time.Now().Unix(),
	}
	if err := FillYoutubeLinkMeta(&link); err != nil {
		return nil, err
	}
	link.LinkID = s.linkRepo.InsertLink(link)
	return &link, nil
}

func (s *ServiceImpl) GetAllLinks() []Link {
	return s.linkRepo.GetAllLinks()
}

func (s *ServiceImpl) GetLinksByUser(userID string) []Link {
	return s.linkRepo.GetLinksByUser(userID)
}

func (s *ServiceImpl) Vote(linkID int64, userID string, score int64) {
	s.voteRepo.MarkVote(linkID, userID, score)
}

func (s *ServiceImpl) GetVotesForUser(links []Link, userID string) map[int64]int64 {
	linkIDs := make([]int64, len(links))
	for i, l := range links {
		linkIDs[i] = l.LinkID
	}
	return s.linkRepo.GetVotesForUser(linkIDs, userID)
}

func (s *ServiceImpl) Test(message string) {
	fmt.Println("Testing message", message)
	s.testRepo.NewTest(message)
}

func (s *ServiceImpl) close() {
	s.voteRepo.close()
	s.userRepo.close()
	s.linkRepo.close()
}
