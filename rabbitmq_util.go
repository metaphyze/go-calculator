package calculator

import (
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"os"
	"sync"
)

var rabbitMqMutex sync.Mutex
var RABBITMQ_HOST string
var RABBITMQ_USERNAME string
var RABBITMQ_PASSWORD string
var RABBITMQ_QUEUE string

func init() {
	RABBITMQ_HOST = os.Getenv("RABBITMQ_HOST")
	RABBITMQ_USERNAME = os.Getenv("RABBITMQ_USERNAME")
	RABBITMQ_PASSWORD = os.Getenv("RABBITMQ_PASSWORD")
	RABBITMQ_QUEUE = os.Getenv("RABBITMQ_QUEUE")
}

type LogEvent struct {
	Username       string  `json:"username"`
	Problem        string  `json:"problem"`
	ID             string  `json:"id"`
	Server         string  `json:"server"`
	RequestNum     uint64  `json:"request_num"`
	StartTime      string  `json:"start_time"`
	StartTimeMs    int64   `json:"start_time_ms"`
	DurationMs     int64   `json:"duration_ms"`
	Success        bool    `json:"success"`
	Error          string  `json:"error"`
	Answer         float64 `json:"answer"`
	HTTPReturnCode int     `json:"http_return_code"`
}

func (le *LogEvent) String() string {
	return fmt.Sprintf("LogEvent{Username: \"%s\", Problem: \"%s\", ID: \"%s\", Server: %s, RequestNum: %d, StartTime: %s, StartTimeMs: %d, DurationMs: %d, Success: %t, Error: \"%s\", Answer: %f, HTTPReturnCode: %d}",
		le.Username, le.Problem, le.ID, le.Server, le.RequestNum, le.StartTime, le.StartTimeMs, le.DurationMs, le.Success, le.Error, le.Answer, le.HTTPReturnCode)
}

func GetInitializedLogEvent() (int64, *LogEvent) {
	startTimeNs, startTimeStr := GetCurrentTimeInHumanReadableDate()
	requestNum := GetRequestNumber()
	logEvent := new(LogEvent)
	logEvent.Username = ""
	logEvent.Problem = ""
	logEvent.ID = ""
	logEvent.Server = ServerId
	logEvent.RequestNum = requestNum
	logEvent.StartTime = startTimeStr
	logEvent.StartTimeMs = startTimeNs / 1000000
	logEvent.DurationMs = 0
	logEvent.Success = false
	logEvent.Answer = 0.0
	logEvent.Error = ""
	logEvent.HTTPReturnCode = 0
	return startTimeNs, logEvent
}

// RabbitMQClient holds the RabbitMQ connection and channel
type RabbitMQClient struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string
}

// NewRabbitMQClient creates a new RabbitMQ connection and channel
func NewRabbitMQClient(rabbitMQHost, queueName string) (*RabbitMQClient, error) {
	// Establish connection with RabbitMQ
	conn, err := amqp.Dial(rabbitMQHost)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	// Open a channel
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %v", err)
	}

	// Declare a queue
	_, err = ch.QueueDeclare(
		queueName, // queue name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare a queue: %v", err)
	}

	client := &RabbitMQClient{
		conn:    conn,
		channel: ch,
		queue:   queueName,
	}

	return client, nil
}

// Close gracefully closes the RabbitMQ connection and channel
func (client *RabbitMQClient) Close() {
	client.channel.Close()
	client.conn.Close()
}

// SendLogEvent sends a log event to RabbitMQ (thread-safe)
func (client *RabbitMQClient) SendLogEvent(logEvent *LogEvent) error {
	// Convert LogEvent to JSON
	eventData, err := json.Marshal(logEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal log event: %v", err)
	}

	rabbitMqMutex.Lock() // Lock for thread safety
	defer rabbitMqMutex.Unlock()

	// Publish the message to the queue
	err = client.channel.Publish(
		"",           // exchange
		client.queue, // routing key (queue name)
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         eventData,
			DeliveryMode: amqp.Persistent, // Ensure message is persistent
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish a message: %v", err)
	}

	return nil
}
