package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ToDo represents a single to-do item.
type ToDo struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Timestamp time.Time `json:"timestamp"`
}

var todos []ToDo
var nextID = 1

func todosHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(todos)
	case http.MethodPost:
		var todo ToDo
		if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		todo.ID = nextID
		nextID++
		todo.Timestamp = time.Now()
		todos = append(todos, todo)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(todo)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func todoHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/todos/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		for _, todo := range todos {
			if todo.ID == id {
				json.NewEncoder(w).Encode(todo)
				return
			}
		}
		http.Error(w, "ToDo not found", http.StatusNotFound)
	case http.MethodPut:
		var updatedTodo ToDo
		if err := json.NewDecoder(r.Body).Decode(&updatedTodo); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for i, todo := range todos {
			if todo.ID == id {
				todos[i].Title = updatedTodo.Title
				todos[i].Timestamp = time.Now()
				json.NewEncoder(w).Encode(todos[i])
				return
			}
		}
		http.Error(w, "ToDo not found", http.StatusNotFound)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func main() {
	http.HandleFunc("/todos", todosHandler)
	http.HandleFunc("/todos/", todoHandler)

	fmt.Println("Server listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
