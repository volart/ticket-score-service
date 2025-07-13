package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

func New(databasePath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}

	if err := db.configure(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to configure database: %w", err)
	}

	if err := db.ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func (db *DB) configure() error {
	// TODO: Make these settings configurable
	db.conn.SetMaxOpenConns(10)
	db.conn.SetMaxIdleConns(5)
	db.conn.SetConnMaxLifetime(time.Hour)
	return nil
}

func (db *DB) ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.conn.PingContext(ctx)
}

func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

func (db *DB) GetConnection() *sql.DB {
	return db.conn
}
