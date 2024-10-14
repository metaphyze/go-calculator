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
	"sync"
	"time"
)

var rabbitMQClientMutex sync.Mutex
var rabbitMQClient *calculator.RabbitMQClient

// Request and Response structures for JSON
type Request struct {
	Username string `json:"username"`
	Problem  string `json:"problem"`
	ID       string `json:"id"`
}

type Response struct {
	Success bool    `json:"success"`
	Error   string  `json:"error"`
	Answer  float64 `json:"answer"`
	ID      string  `json:"id"`
}

func setupRabbitMqConnection() {
	rabbitMQClientMutex.Lock()
	defer rabbitMQClientMutex.Unlock()

	rabbitMQHost := fmt.Sprintf("amqp://%v:%v@%v:5672/",
		calculator.RABBITMQ_USERNAME, calculator.RABBITMQ_PASSWORD, calculator.RABBITMQ_HOST)

	// Create RabbitMQ client
	var err error
	rabbitMQClient, err = calculator.NewRabbitMQClient(rabbitMQHost, calculator.RABBITMQ_QUEUE)

	if err != nil {
		log.Fatalf("Failed to create RabbitMQ client: %v", err)
	}
}

func sendLogEvent(logEvent *calculator.LogEvent) {
	rabbitMQClientMutex.Lock()
	defer rabbitMQClientMutex.Unlock()

	// Send log event
	err := rabbitMQClient.SendLogEvent(logEvent)
	if err != nil {
		log.Printf("Error sending log event: %v", err)
	} else {
		log.Printf("Sent RabbitMQ message: %+v", logEvent)
	}
}

func handleCalculation(w http.ResponseWriter, r *http.Request) {
	startTimeNs, logEvent := calculator.GetInitializedLogEvent()
	defer func() {
		logEvent.DurationMs = (time.Now().UTC().UnixNano() - startTimeNs) / 1000000
		go sendLogEvent(logEvent)
	}()

	if r.Method != http.MethodPost {
		logEvent.Error = "Invalid request method"
		logEvent.HTTPReturnCode = http.StatusMethodNotAllowed
		http.Error(w, logEvent.Error, logEvent.HTTPReturnCode)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logEvent.Error = "Invalid request body"
		logEvent.HTTPReturnCode = http.StatusBadRequest
		http.Error(w, logEvent.Error, logEvent.HTTPReturnCode)
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

	logEvent.Success = resp.Success
	logEvent.Error = resp.Error
	logEvent.Answer = resp.Answer
	logEvent.ID = req.ID
	logEvent.Username = req.Username
	logEvent.Problem = req.Problem
	logEvent.HTTPReturnCode = http.StatusOK

	jsonData, err := json.Marshal(resp)
	if err != nil {
		logEvent.Error = "Error encoding response"
		logEvent.HTTPReturnCode = http.StatusInternalServerError
		http.Error(w, logEvent.Error, logEvent.HTTPReturnCode)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, writeErr := w.Write(jsonData)
	if writeErr != nil {
		logEvent.Success = false
		logEvent.Error = fmt.Sprintf("Error writing response: %v", err)
		logEvent.Answer = 0
		logEvent.ID = req.ID
		logEvent.Username = req.Username
		logEvent.Problem = req.Problem
		logEvent.HTTPReturnCode = http.StatusInternalServerError
	}
}

/*
curl -X POST http://localhost:8080/calculate \
-H "Content-Type: application/json" \
-d '{
"problem": "6/2",
"id": "1234",
"username": "user1"
}'
*/

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
	setupRabbitMqConnection()

	http.HandleFunc("/calculate", handleCalculation)

	fmt.Printf("Server starting on port %s\n", *port)
	if err := http.ListenAndServe(address, nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
