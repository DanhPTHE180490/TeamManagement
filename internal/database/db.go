package database

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql" // The blank identifier imports the driver without using its functions directly
)

// InitDB opens a connection pool to MySQL
func InitDB() *sql.DB {
	// The Data Source Name (DSN) format: username:password@tcp(host:port)/dbname?parseTime=true
	// Replace "root" and "password" with your local DataGrip credentials
	dsn := "root:password_1234@tcp(127.0.0.1:3306)/microservices_capstone?parseTime=true"

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	// Connection pool settings (Best practices for microservices)
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Successfully connected to MySQL!")
	return db
}
