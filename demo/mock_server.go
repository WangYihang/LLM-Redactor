package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusInternalServerError)
			return
		}
		defer func() { _ = r.Body.Close() }()

		fmt.Println("\n[PROVIDER RECEIVED DATA]:")
		var prettyJSON interface{}
		if err := json.Unmarshal(body, &prettyJSON); err == nil {
			formatted, _ := json.MarshalIndent(prettyJSON, "", "  ")
			fmt.Println(string(formatted))
		} else {
			fmt.Println(string(body))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"choices": [{"message": {"content": "Hello! I am a mock AI. Your request was received and logged."}}]}`)
	})

	log.Println("Mock Provider starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
