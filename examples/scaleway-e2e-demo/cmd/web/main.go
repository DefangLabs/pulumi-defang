package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type apiResponse struct {
	Service string `json:"service"`
	From    string `json:"from"`
	Count   int    `json:"count"`
	Message string `json:"message"`
}

func main() {
	apiURL := strings.TrimRight(os.Getenv("API_URL"), "/")
	if apiURL == "" {
		log.Fatal("API_URL is required")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	client := &http.Client{Timeout: 15 * time.Second}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, status, err := callAPI(client, apiURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		var decoded apiResponse
		_ = json.Unmarshal(body, &decoded)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!doctype html>
<html>
<head><title>Scaleway Defang demo</title></head>
<body>
<h1>Scaleway Defang demo</h1>
<p><strong>web</strong> called <strong>api</strong>, and api wrote to managed Postgres.</p>
<p>API status: %d</p>
<p>Database hit count: %d</p>
<pre>%s</pre>
</body>
</html>`, status, decoded.Count, html.EscapeString(string(body)))
	})

	log.Printf("web listening on :%s; api=%s", port, apiURL)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func callAPI(client *http.Client, apiURL string) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, apiURL+"/", nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("X-Defang-Demo-From", "web")
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("call api: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return body, resp.StatusCode, fmt.Errorf("api returned %d: %s", resp.StatusCode, string(body))
	}
	return body, resp.StatusCode, nil
}
