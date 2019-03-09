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
	close()
}

type ServiceImpl struct {
	linkRepo LinkRepository
	userRepo UserRepository
	voteRepo VoteRepository
	testRepo TestRepository
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
		TotalVotes:  0,
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

func (s *ServiceImpl) Vote(linkID int64, userID string, score int64) {
	s.voteRepo.MarkVote(linkID, userID, score)
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
