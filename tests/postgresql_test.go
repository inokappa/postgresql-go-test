package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"os"
	"testing"
)

var (
	host     = os.Getenv("DB_HOST")
	user     = os.Getenv("DB_USER")
	password = os.Getenv("DB_PASSWORD")
	dbname   = os.Getenv("DB_NAME")
)

func TestPostgrePingWithSSL(t *testing.T) {
	db, err := dbConnection(true)
	if err != nil {
		t.Errorf("got error: %v", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		t.Errorf("got error: %v", err)
	}
}

func TestPostgrePingWithoutSSL(t *testing.T) {
	db, err := dbConnection(false)
	if err != nil {
		t.Errorf("got error: %v", err)
	}
	defer db.Close()

	err = db.Ping()
	if err == nil {
		t.Errorf("got nil: Should not be able to connect.")
	} else {
		t.Log(err)
	}
}
func dbConnection(ssl bool) (*sql.DB, error) {
	var ssl_mode string
	if ssl {
		ssl_mode = "require"
	} else {
		ssl_mode = "disable"
	}
	connInfo := fmt.Sprintf("host=%s "+
		"port=5432 "+
		"user=%s "+
		"password=%s "+
		"dbname=%s "+
		"sslmode=%s",
		host, user, password, dbname, ssl_mode)
	db, err := sql.Open("postgres", connInfo)
	if err != nil {
		return nil, err
	}

	return db, nil
}
