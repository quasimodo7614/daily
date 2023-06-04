package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

const timeFormat = "2006-01-02T15:04:05Z07:00"

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
	http.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			getStatickItems(db, w, r)
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

func convertdata(items []CompletedItem) (map[string]interface{}, error) {
	//// 按照日期分组
	//b, _ := json.Marshal(items)
	//log.Println("data is: ", string(b))
	groups := make(map[string][]CompletedItem)
	for _, item := range items {
		// 解析 CompletedTime 字段，转换为 time.Time 类型的值
		completedTime, err := time.Parse(timeFormat, item.CompletedTime)
		if err != nil {
			fmt.Println("Error parsing time:", err)
			continue
		}

		// 根据日期生成标签
		label := completedTime.Format("01/02")

		// 将 CompletedItem 添加到对应日期的数组中
		groups[label] = append(groups[label], item)
	}

	// 对日期标签进行排序
	labels := make([]string, 0, len(groups))

	milk := map[string]int{}
	diapers := map[string]int{}
	poops := map[string]int{}

	for label, items := range groups {
		labels = append(labels, label)

		milkCount := 0
		diapersCount := 0
		poopsCount := 0
		for _, item := range items {
			if item.Item == "喂奶" {
				// 剔除字符串中的 "ml"，并将剩余部分转换为整数
				value, err := strconv.Atoi(strings.TrimSuffix(item.Description, " ml"))
				if err != nil {
					fmt.Println("Error converting string to int:", err)
					continue
				}
				milkCount += value

			} else if item.Item == "尿布湿" {
				diapersCount++
			} else if item.Item == "大便" {
				poopsCount++
			}
		}
		milk[label] = milkCount
		diapers[label] = diapersCount
		poops[label] = poopsCount
	}
	sort.Strings(labels)

	milkArray := make([]int, 0, len(groups))
	diaperArray := make([]int, 0, len(groups))
	poopsArray := make([]int, 0, len(groups))

	for _, label := range labels {
		milkArray = append(milkArray, milk[label])
		diaperArray = append(diaperArray, diapers[label])
		poopsArray = append(poopsArray, poops[label])
	}

	// 生成 JSON 对象
	return map[string]interface{}{
		"labels":  labels,
		"milk":    milk,
		"diapers": diapers,
		"poops":   poops,
	}, nil

}
func getStatickItems(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days, err := strconv.Atoi(daysStr)
	if err != nil {
		http.Error(w, "Invalid days parameter", http.StatusBadRequest)
		return
	}
	sub := -(days - 1)
	today := time.Now().AddDate(0, 0, sub)
	todayStr := today.Format("2006-01-02")
	rows, err := db.Query("SELECT * FROM completed_items WHERE completed_time > $1 ORDER BY id DESC", todayStr)
	if err != nil {
		log.Println("query err:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	completedItems := []CompletedItem{}
	for rows.Next() {
		completedItem := CompletedItem{}
		err := rows.Scan(&completedItem.ID, &completedItem.Item, &completedItem.Description, &completedItem.CompletedTime)
		if err != nil {
			log.Println("scan err:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		completedItems = append(completedItems, completedItem)
	}

	/*
		{
		  "labels": ["05/29", "05/30", "05/31", "06/01", "06/02", "06/03", "06/04"],
		  "milk": [300, 400, 600, 700, 800, 900, 1000],
		  "diapers": [3, 4, 5, 6, 5, 4, 3],
		  "poops": [1, 2, 3, 2, 1, 0, 1]
		}
	*/

	data, err := convertdata(completedItems)
	if err != nil {
		log.Println("convertdata err:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
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
