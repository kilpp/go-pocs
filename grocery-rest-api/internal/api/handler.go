package api

import (
	"encoding/json"
	"net/http"

	"grocery-rest-api/internal/model"
	"grocery-rest-api/internal/service"
)

type Handler struct {
	groceryService *service.GroceryService
}

func NewHandler(gs *service.GroceryService) *Handler {
	return &Handler{groceryService: gs}
}

func (h *Handler) AddItemHandler(w http.ResponseWriter, r *http.Request) {
	var item model.GroceryItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err := h.groceryService.AddItem(item.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) RemoveItemHandler(w http.ResponseWriter, r *http.Request) {
	var item model.GroceryItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err := h.groceryService.RemoveItem(item.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) MarkAsDoneHandler(w http.ResponseWriter, r *http.Request) {
	var item model.GroceryItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err := h.groceryService.MarkAsDone(item.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ListItemsHandler(w http.ResponseWriter, r *http.Request) {
	items, err := h.groceryService.ListItems()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}