package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" // PostgreSQL driver
	"grocery-rest-api/internal/model"
)

type DB struct {
	Conn *sql.DB
}

func NewDB(dataSourceName string) (*DB, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{Conn: db}, nil
}

func (db *DB) SaveGroceryList(groceryList []model.GroceryItem) error {
	// Implement logic to save grocery list to the database
	for _, item := range groceryList {
		_, err := db.Conn.Exec("INSERT INTO grocery_items (name, done) VALUES ($1, $2)", item.Name, item.Done)
		if err != nil {
			return fmt.Errorf("failed to insert item %s: %w", item.Name, err)
		}
	}
	return nil
}

func (db *DB) LoadGroceryList() ([]model.GroceryItem, error) {
	var groceryList []model.GroceryItem
	rows, err := db.Conn.Query("SELECT name, done FROM grocery_items")
	if err != nil {
		return nil, fmt.Errorf("failed to query grocery items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item model.GroceryItem
		if err := rows.Scan(&item.Name, &item.Done); err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		groceryList = append(groceryList, item)
	}

	return groceryList, nil
}

func (db *DB) Close() error {
	return db.Conn.Close()
}