package web

import (
	"fmt"
	"net/http"
	"sharded-kvs/db"
)

// contains HTTP method handlers to be used for GETting and SETting
type Server struct {
	db *db.Database
}

// creates a new instance with HTTP handlers for GET and SET operations
func NewServer(db *db.Database) *Server {
	return &Server{
		db: db,
	}
}

// handles read requests from the DB
func (s *Server) GetHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	key := r.Form.Get("key")
	value, err := s.db.GetKey(key)
	fmt.Fprintf(w, "Value = %q, Error = %v", value, err)
}

// handles WRITE requests from the DB
func (s *Server) SetHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	key := r.Form.Get("key")
	value := r.Form.Get("value")
	err := s.db.SetKey(key, []byte(value))
	fmt.Fprintf(w, "Error = %v", err)
}
