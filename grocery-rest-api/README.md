# Grocery REST API

This project is a RESTful API for managing a grocery list. It allows users to add, remove, mark items as done, and list grocery items. The grocery list can be stored in a JSON file and will later support database storage.

## Project Structure

```
grocery-rest-api
├── cmd
│   └── main.go          # Entry point of the application
├── internal
│   ├── api
│   │   └── handler.go   # HTTP handlers for the grocery list API
│   ├── model
│   │   └── grocery.go    # Defines the GroceryItem struct
│   ├── storage
│   │   ├── file.go      # Storage functionality for file-based storage
│   │   └── db.go        # Storage functionality for database storage
│   └── service
│       └── grocery.go    # Business logic for managing the grocery list
├── go.mod                # Module definition
├── go.sum                # Module dependency checksums
└── README.md             # Project documentation
```

## Setup Instructions

1. Clone the repository:
   ```
   git clone <repository-url>
   cd grocery-rest-api
   ```

2. Install the necessary dependencies:
   ```
   go mod tidy
   ```

3. Run the application:
   ```
   go run cmd/main.go
   ```

## Usage

### Endpoints

- **Add Item**
  - `POST /grocery`
  - Body: `{ "name": "item_name" }`

- **Remove Item**
  - `DELETE /grocery`
  - Body: `{ "name": "item_name" }`

- **Mark Item as Done**
  - `PUT /grocery/done`
  - Body: `{ "name": "item_name" }`

- **Mark Item as Not Done**
  - `PUT /grocery/redo`
  - Body: `{ "name": "item_name" }`

- **List All Items**
  - `GET /grocery`

## Future Enhancements

- Implement database storage for grocery items.
- Add user authentication and authorization.
- Improve error handling and validation.