package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/mnogu/go-calculator"
	"log"
	"math"
	"net/http"
	"os"
)

// Request and Response structures for JSON
type Request struct {
	Problem string `json:"problem"`
	ID      string `json:"id"`
}

type Response struct {
	Success bool    `json:"success"`
	Error   string  `json:"error"`
	Answer  float64 `json:"answer"`
	ID      string  `json:"id"`
}

func handleCalculation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Evaluate the mathematical expression
	result, err := calculator.Calculate(req.Problem)

	resp := Response{
		ID: req.ID,
	}

	// Handle success and errors
	if err != nil {
		resp.Success = false
		resp.Error = err.Error()
		resp.Answer = 0
	} else {
		if err == nil && math.IsInf(result, 1) {
			resp.Success = false
			resp.Error = "+infinity"
			resp.Answer = 0
		} else if err == nil && math.IsInf(result, -1) {
			resp.Success = false
			resp.Error = "-infinity"
			resp.Answer = 0
		} else if err == nil && math.IsNaN(result) {
			resp.Success = false
			resp.Error = "NaN"
			resp.Answer = 0
		} else {
			resp.Success = true
			resp.Error = ""
			resp.Answer = result
		}
	}

	// Set content type and write the JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

func main() {
	// Use flag package to allow user to specify port
	port := flag.String("port", "8080", "port to listen on")
	flag.Parse()

	// Check if PORT environment variable is set
	envPort := os.Getenv("PORT")
	if envPort != "" {
		*port = envPort
	}

	address := ":" + *port

	http.HandleFunc("/calculate", handleCalculation)

	fmt.Printf("Server starting on port %s\n", *port)
	if err := http.ListenAndServe(address, nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
