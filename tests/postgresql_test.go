package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"os"
	"testing"
)

const (
	user   = "pgadmin"
	dbname = "postgres"
)

var (
	host     = os.Getenv("DB_HOST")
	password = os.Getenv("DB_PASSWORD")
)

func TestPostgrePing(t *testing.T) {
	// connInfo := fmt.Sprintf("host=%s "+
	// 	"port=5432 "+
	// 	"user=%s "+
	// 	"password=%s "+
	// 	"dbname=%s "+
	// 	"sslmode=disable",
	// 	host, user, password, dbname)
	// db, err := sql.Open("postgres", connInfo)
	// if err != nil {
	// 	t.Errorf("got error: %v", err)
	// }
	db, err := dbConnection()
	if err != nil {
		t.Errorf("got error: %v", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		t.Errorf("got error: %v", err)
	}
}

func dbConnection() (*sql.DB, error) {
	connInfo := fmt.Sprintf("host=%s "+
		"port=5432 "+
		"user=%s "+
		"password=%s "+
		"dbname=%s "+
		"sslmode=disable",
		host, user, password, dbname)
	db, err := sql.Open("postgres", connInfo)
	if err != nil {
		return nil, err
	}

	return db, nil
}
