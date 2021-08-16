package main

type PersonService interface {
	Find(int) ([]Person, error)
}

type personService struct {
	repository PersonRepository
}

// interfaceを実装しているか保証する
var _ PersonService = (*personService)(nil)

func NewPersonService(repository PersonRepository) PersonService {
	return &personService{repository: repository}
}

func (s *personService) Find(id int) ([]Person, error) {
	return s.repository.FindByID(id)
}
