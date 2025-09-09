package service

import (
	"sync"

	"grocery-rest-api/internal/model"
	"grocery-rest-api/internal/storage"
)

type GroceryService struct {
	storage storage.Storage
	mu      sync.Mutex
}

func NewGroceryService(storage storage.Storage) *GroceryService {
	return &GroceryService{storage: storage}
}

func (s *GroceryService) AddItem(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item := model.GroceryItem{Name: name, Done: false}
	err := s.storage.SaveItem(item)
	if err != nil {
		return err
	}
	return nil
}

func (s *GroceryService) RemoveItem(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.storage.RemoveItem(name)
	if err != nil {
		return err
	}
	return nil
}

func (s *GroceryService) MarkAsDone(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, err := s.storage.GetItem(name)
	if err != nil {
		return err
	}
	item.Done = true
	return s.storage.UpdateItem(item)
}

func (s *GroceryService) MarkAsNotDone(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, err := s.storage.GetItem(name)
	if err != nil {
		return err
	}
	item.Done = false
	return s.storage.UpdateItem(item)
}

func (s *GroceryService) ListItems() ([]model.GroceryItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.storage.GetAllItems()
}