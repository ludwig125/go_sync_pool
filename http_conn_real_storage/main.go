package main

import (
	"fmt"
	"log"
)

func main() {
	if err := runPersonAPI("8080", "sample_db"); err != nil {
		log.Panicf("failed to runPersonAPI: %v", err)
	}
}

func runPersonAPI(serverPort, dbName string) error {
	var repository PersonRepository
	var err error

	log.Println("use sqlite. database name:", dbName)
	repository, err = NewSQLitePersonRepository(dbName)
	if err != nil {
		return fmt.Errorf("failed to NewMySQLPersonRepository: %v", err)
	}

	service := NewPersonService(repository)

	server := NewServer(serverPort, service)
	return server.Run()
}
