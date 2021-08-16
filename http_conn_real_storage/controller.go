package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

type Server struct {
	Port    string
	Service PersonService
}

func NewServer(serverPort string, service PersonService) *Server {
	return &Server{
		Port:    serverPort,
		Service: service,
	}
}

func (s *Server) FindHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("%s failed to conv to int", idStr)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ps, err := s.Service.Find(id)
	if err != nil {
		e := fmt.Sprintf("failed to Find: %v", err)
		log.Println(e)
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, e, http.StatusInternalServerError)
		return
	}
	jsonData, err := json.Marshal(ps)
	if err != nil {
		log.Println("failed to marshal", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, string(jsonData))
}

func (s *Server) Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.FindHandler)
	srv := &http.Server{
		Addr:    "localhost:" + s.Port,
		Handler: mux,
	}
	fmt.Println("starting http server on :", s.Port)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("Server closed with error: %v", err)
	}
	return nil
}
