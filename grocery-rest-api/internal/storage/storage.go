package storage

import "grocery-rest-api/internal/model"

type Storage interface {
	SaveItem(item model.GroceryItem) error
	RemoveItem(name string) error
	GetItem(name string) (model.GroceryItem, error)
	UpdateItem(item model.GroceryItem) error
	GetAllItems() ([]model.GroceryItem, error)
}
