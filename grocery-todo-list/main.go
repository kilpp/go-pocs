package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type GroceryItem struct {
	Name string
	Done bool
}

var groceryList []GroceryItem

const dataFile = "grocery.json"

func loadList() {
	data, err := ioutil.ReadFile(dataFile)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, start with an empty list
			groceryList = []GroceryItem{}
			return
		}
		fmt.Printf("Error loading list: %v\n", err)
		os.Exit(1)
	}

	err = json.Unmarshal(data, &groceryList)
	if err != nil {
		fmt.Printf("Error unmarshaling list: %v\n", err)
		os.Exit(1)
	}
}

func saveList() {
	data, err := json.MarshalIndent(groceryList, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling list: %v\n", err)
		os.Exit(1)
	}

	err = ioutil.WriteFile(dataFile, data, 0644)
	if err != nil {
		fmt.Printf("Error saving list: %v\n", err)
		os.Exit(1)
	}
}

func addItem(name string) {
	groceryList = append(groceryList, GroceryItem{Name: name, Done: false})
	fmt.Printf("Added: %s\n", name)
	saveList()
}

func removeItem(name string) {
	found := false
	for i, item := range groceryList {
		if item.Name == name {
			groceryList = append(groceryList[:i], groceryList[i+1:]...)
			fmt.Printf("Removed: %s\n", name)
			found = true
			break
		}
	}
	if !found {
		fmt.Printf("Item not found: %s\n", name)
	}
	saveList()
}

func markAsDone(name string) {
	found := false
	for i, item := range groceryList {
		if item.Name == name {
			groceryList[i].Done = true
			fmt.Printf("Done: %s\n", name)
			found = true
			break
		}
	}
	if !found {
		fmt.Printf("Item not found: %s\n", name)
	}
	saveList()
}

func markAsNotDone(name string) {
	found := false
	for i, item := range groceryList {
		if item.Name == name {
			groceryList[i].Done = false
			fmt.Printf("Redo: %s\n", name)
			found = true
			break
		}
	}
	if !found {
		fmt.Printf("Item not found: %s\n", name)
	}
	saveList()
}

func listAllItems() {
	if len(groceryList) == 0 {
		fmt.Println("Grocery list is empty.")
		return
	}
	fmt.Println("Grocery List:")
	for _, item := range groceryList {
		status := " "
		if item.Done {
			status = "x"
		}
		fmt.Printf("[%s] %s\n", status, item.Name)
	}
}

func main() {
	loadList()

	if len(os.Args) < 2 {
		fmt.Println("Usage: ./grocery-todo <command> [arguments]")
		return
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "add":
		if len(args) > 0 {
			addItem(strings.Join(args, " "))
		} else {
			fmt.Println("Please provide an item to add.")
		}
	case "remove":
		if len(args) > 0 {
			removeItem(strings.Join(args, " "))
		} else {
			fmt.Println("Please provide an item to remove.")
		}
	case "done":
		if len(args) > 0 {
			markAsDone(strings.Join(args, " "))
		} else {
			fmt.Println("Please provide an item to mark as done.")
		}
	case "redo":
		if len(args) > 0 {
			markAsNotDone(strings.Join(args, " "))
		} else {
			fmt.Println("Please provide an item to mark as not done.")
		}
	case "list":
		listAllItems()
	default:
		fmt.Println("Unknown command.")
	}
}
