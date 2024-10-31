package db

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

type Url struct {
	Id   int
	Name string
	Url  string
}

var db *pgx.Conn

// Connect to the database
func Connect() (*pgx.Conn, error) {
	connStr := os.Getenv("DATABASE_URL")
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		return nil, err
	}
	db = conn
	return conn, nil
}

// Create table url if not exists
func CreateTable() error {
	_, err := db.Exec(context.Background(), "CREATE TABLE IF NOT EXISTS url (id SERIAL PRIMARY KEY, name TEXT UNIQUE, url TEXT)")
	return err
}

// Insert a new record into the database
func Insert(name, url string) error {
	_, err := db.Exec(context.Background(), "INSERT INTO url (name, url) VALUES ($1, $2)", name, url)
	return err
}

// Update a record in the database
func Update(name, url string) error {
	_, err := db.Exec(context.Background(), "UPDATE url SET url = $2 WHERE name = $1", name, url)
	return err
}

// Delete a record from the database
func Delete(name string) error {
	_, err := db.Exec(context.Background(), "DELETE FROM url WHERE name = $1", name)
	return err
}

// List all records from the database
func List() (map[string]Url, error) {
	rows, err := db.Query(context.Background(), "SELECT name, url FROM url")
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer rows.Close()

	urls := make(map[string]Url)
	for rows.Next() {
		var url Url
		err := rows.Scan(&url.Name, &url.Url)
		if err != nil {
			return nil, err
		}
		urls[url.Name] = url
	}
	return urls, nil
}

// Get a record from the database
func Get(name string) (string, error) {
	var url string
	err := db.QueryRow(context.Background(), "SELECT url FROM url WHERE name = $1", name).Scan(&url)
	if err != nil {
		return "", err
	}
	return url, nil
}

// Close the database connection
func Close(conn *pgx.Conn) {
	conn.Close(context.Background())
}
