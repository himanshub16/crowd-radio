package main

import (
	"fmt"
)

type Service interface {
	SubmitLink(url string) *Link
	Vote(linkID uint64, score uint64) uint64
	Test(message string)
	close()
}

type ServiceImpl struct {
	linkRepo LinkRepository
	userRepo UserRepository
	voteRepo VoteRepository
	testRepo TestRepository
}

func (s *ServiceImpl) SubmitLink(url string) *Link {
	// required checks here
	// s.linkRepo.InsertLink(somethinghere)
	fmt.Println("submitted", url)
	return nil
}

func (s *ServiceImpl) Vote(linkID uint64, scrore uint64) uint64 {
	return 0
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
