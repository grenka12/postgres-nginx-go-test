package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func connect() (*sql.DB, error) {
	bin, err := ioutil.ReadFile("/run/secrets/db-password")
	if err != nil {
		return nil, err
	}
	return sql.Open("postgres", fmt.Sprintf("postgres://postgres:%s@postgres:5432/example?sslmode=disable", string(bin)))
}

// GET / -> список блогпостів
func blogHandler(w http.ResponseWriter, r *http.Request) {
	db, err := connect()
	if err != nil {
		w.WriteHeader(500)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, title FROM blog ORDER BY id")
	if err != nil {
		w.WriteHeader(500)
		return
	}
	defer rows.Close()

	type Blog struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	}

	var blogs []Blog
	for rows.Next() {
		var b Blog
		if err := rows.Scan(&b.ID, &b.Title); err != nil {
			w.WriteHeader(500)
			return
		}
		blogs = append(blogs, b)
	}

	json.NewEncoder(w).Encode(blogs)
}

// POST /add -> додати новий блогпост
func addBlogHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title string `json:"title"`
	}


	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(400)
		return
	}

	db, err := connect()
		if err != nil {
			log.Printf("DB connect error: %v", err)
			w.WriteHeader(500)
			fmt.Fprintf(w, "DB connect error: %v", err)
			return
		}
	defer db.Close()

	_, err = db.Exec("INSERT INTO blog (title) VALUES ($1)", input.Title)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(201)
	fmt.Fprintf(w, "Blog post added: %s", input.Title)
}

func main() {
	log.Print("Prepare db...")
	if err := prepare(); err != nil {
		log.Fatal(err)
	}

	log.Print("Listening 8000")
	r := mux.NewRouter()
	r.HandleFunc("/", blogHandler).Methods("GET")
	r.HandleFunc("/add", addBlogHandler).Methods("POST")
	log.Fatal(http.ListenAndServe(":8000", handlers.LoggingHandler(os.Stdout, r)))

	r.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("Request %s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
})

}

func prepare() error {
	db, err := connect()
	if err != nil {
		return err
	}
	defer db.Close()

	for i := 0; i < 60; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	// Створюємо таблицю, якщо її ще нема
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS blog (id SERIAL PRIMARY KEY, title VARCHAR)")
	if err != nil {
		return err
	}

	// Додаємо стартові пости лише якщо таблиця порожня
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM blog").Scan(&count)
	if count == 0 {
		for i := 0; i < 5; i++ {
			if _, err := db.Exec("INSERT INTO blog (title) VALUES ($1);", fmt.Sprintf("Blog post #%d", i)); err != nil {
				return err
			}
		}
	}

	return nil
}
