package main

type PersonRepository interface {
	FindByID(int) ([]Person, error)
}
