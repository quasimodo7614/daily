package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

type CompletedItem struct {
	ID            int    `json:"id"`
	Item          string `json:"item"`
	Description   string `json:"description"`
	CompletedTime string `json:"completed_time"`
}

func getdburl() string {
	if s := os.Getenv("PG_URL"); s != "" {
		return s
	}
	return "user=postgres password=123456 host=localhost port=5432 dbname=zze sslmode=disable"
}
func main() {
	db, err := sql.Open("postgres", getdburl())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./index.html")
	})
	http.HandleFunc("/completed-items", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			getCompletedItems(db, w, r)
		case "POST":
			addCompletedItem(db, w, r)
		case http.MethodDelete:
			deleteCompletedItem(db, w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Listening on :%s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
func getCompletedItems(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	today := time.Now().Format("2006-01-02")
	rows, err := db.Query("SELECT * FROM completed_items WHERE completed_time > $1 ORDER BY id DESC", today)
	if err != nil {
		log.Println("query err:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	completedItems := []CompletedItem{}
	for rows.Next() {
		completedItem := CompletedItem{}
		var completedTimeStr string
		err := rows.Scan(&completedItem.ID, &completedItem.Item, &completedItem.Description, &completedTimeStr)
		if err != nil {
			log.Println("scan err:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		completedTime, err := time.Parse("2006-01-02T15:04:05Z07:00", completedTimeStr)
		if err != nil {
			log.Println("parse time err:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		completedItem.CompletedTime = completedTime.Format("15:04")
		completedItems = append(completedItems, completedItem)
	}

	sort.Slice(completedItems, func(i, j int) bool {
		return completedItems[i].CompletedTime > completedItems[j].CompletedTime
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(completedItems)
}

func addCompletedItem(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	var completedItem CompletedItem
	err := json.NewDecoder(r.Body).Decode(&completedItem)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if completedItem.CompletedTime == "" {
		completedItem.CompletedTime = time.Now().Format("2006-01-02 15:04")
	}

	_, err = db.Exec("INSERT INTO completed_items(item, description, completed_time) VALUES($1, $2, $3)", completedItem.Item, completedItem.Description, completedItem.CompletedTime)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(completedItem)
}

func deleteCompletedItem(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	log.Println("delete item")
	// Extract the id parameter from the URL path
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Println("Invalid ID parameter")
		http.Error(w, "Invalid ID parameter", http.StatusBadRequest)
		return
	}

	result, err := db.Exec("DELETE FROM completed_items WHERE id = $1", id)
	if err != nil {
		log.Println("db err:", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println("db err:", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		log.Println("not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
