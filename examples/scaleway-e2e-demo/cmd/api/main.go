package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
)

type response struct {
	Service string `json:"service"`
	From    string `json:"from,omitempty"`
	Count   int    `json:"count"`
	Message string `json:"message"`
}

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		count, err := recordHit(ctx, databaseURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response{
			Service: "api",
			From:    r.Header.Get("X-Defang-Demo-From"),
			Count:   count,
			Message: "api wrote a row to Postgres and read the total count",
		})
	})

	log.Printf("api listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func recordHit(ctx context.Context, databaseURL string) (int, error) {
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return 0, fmt.Errorf("connect to postgres: %w", err)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, `
		create table if not exists demo_hits (
			id bigserial primary key,
			created_at timestamptz not null default now()
		)
	`); err != nil {
		return 0, fmt.Errorf("create table: %w", err)
	}
	if _, err := conn.Exec(ctx, `insert into demo_hits default values`); err != nil {
		return 0, fmt.Errorf("insert hit: %w", err)
	}
	var count int
	if err := conn.QueryRow(ctx, `select count(*) from demo_hits`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count hits: %w", err)
	}
	return count, nil
}
