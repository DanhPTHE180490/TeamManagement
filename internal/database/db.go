package database

import (
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// InitDB opens a connection pool to MySQL
func InitDB() *sqlx.DB {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = os.Getenv("DB_USER") + ":" + os.Getenv("DB_PASSWORD") + "@tcp(127.0.0.1:3306)/" + os.Getenv("DB_NAME") + "?parseTime=true"
		log.Println("Warning: DB_DSN environment variable not found. Using local fallback.")
	}
	// Connect does both sqlx.Connect() AND db.Ping()
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	// Connection pool settings (Best practices for microservices)
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Successfully connected to MySQL!")
	return db
}
