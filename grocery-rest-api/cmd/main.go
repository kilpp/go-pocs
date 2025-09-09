package main

import (
	"log"
	"net/http"

	"grocery-rest-api/internal/api"
	"grocery-rest-api/internal/service"
	"grocery-rest-api/internal/storage"
)

func main() {
	fileStorage, err := storage.NewFileStorage()
	if err != nil {
		log.Fatalf("Failed to create file storage: %v", err)
	}

	groceryService := service.NewGroceryService(fileStorage)
	handler := api.NewHandler(groceryService)

	http.HandleFunc("/items", handler.ListItemsHandler)
	http.HandleFunc("/items/add", handler.AddItemHandler)
	http.HandleFunc("/items/remove", handler.RemoveItemHandler)
	http.HandleFunc("/items/done", handler.MarkAsDoneHandler)

	log.Println("Server running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}