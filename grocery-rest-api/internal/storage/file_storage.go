package storage

import (
	"encoding/json"
	"errors"
	"os"
	"sync"

	"grocery-rest-api/internal/model"
)

const dataFile = "grocery.json"

type FileStorage struct {
	mu    sync.Mutex
	items []model.GroceryItem
}

func NewFileStorage() (*FileStorage, error) {
	fs := &FileStorage{}
	err := fs.load()
	if err != nil {
		return nil, err
	}
	return fs, nil
}

func (fs *FileStorage) load() error {
	data, err := os.ReadFile(dataFile)
	if err != nil {
		if os.IsNotExist(err) {
			fs.items = []model.GroceryItem{}
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &fs.items)
}

func (fs *FileStorage) save() error {
	data, err := json.MarshalIndent(fs.items, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(dataFile, data, 0644)
}

func (fs *FileStorage) SaveItem(item model.GroceryItem) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for i, existingItem := range fs.items {
		if existingItem.Name == item.Name {
			fs.items[i] = item
			return fs.save()
		}
	}
	fs.items = append(fs.items, item)
	return fs.save()
}

func (fs *FileStorage) RemoveItem(name string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for i, item := range fs.items {
		if item.Name == name {
			fs.items = append(fs.items[:i], fs.items[i+1:]...)
			return fs.save()
		}
	}
	return errors.New("item not found")
}

func (fs *FileStorage) GetItem(name string) (model.GroceryItem, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for _, item := range fs.items {
		if item.Name == name {
			return item, nil
		}
	}
	return model.GroceryItem{}, errors.New("item not found")
}

func (fs *FileStorage) UpdateItem(item model.GroceryItem) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for i, existingItem := range fs.items {
		if existingItem.Name == item.Name {
			fs.items[i] = item
			return fs.save()
		}
	}
	return errors.New("item not found")
}

func (fs *FileStorage) GetAllItems() ([]model.GroceryItem, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Return a copy to prevent external modification
	itemsCopy := make([]model.GroceryItem, len(fs.items))
	copy(itemsCopy, fs.items)
	return itemsCopy, nil
}
