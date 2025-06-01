package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Client represents a connected SSE client
type Client struct {
	id     string
	events chan string
}

// EventServer manages SSE connections and broadcasts
type EventServer struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan string
	mutex      sync.RWMutex
}

// NewEventServer creates a new EventServer instance
func NewEventServer() *EventServer {
	return &EventServer{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan string),
	}
}

func (es *EventServer) run() {
	for {
		select {
		case client := <-es.register:
			es.mutex.Lock()
			es.clients[client.id] = client
			es.mutex.Unlock()
			log.Printf("Client connected: %s\n", client.id)

		case client := <-es.unregister:
			es.mutex.Lock()
			if _, ok := es.clients[client.id]; ok {
				delete(es.clients, client.id)
				close(client.events)
			}
			es.mutex.Unlock()
			log.Printf("Client disconnected: %s\n", client.id)

		case message := <-es.broadcast:
			es.mutex.RLock()
			for _, client := range es.clients {
				select {
				case client.events <- message:
				default:
					// Skip if client's channel is full
				}
			}
			es.mutex.RUnlock()
		}
	}
}

func (es *EventServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a new client
	client := &Client{
		id:     fmt.Sprintf("%d", time.Now().UnixNano()),
		events: make(chan string, 10),
	}

	// Register client
	es.register <- client

	// Ensure client cleanup on disconnect
	defer func() {
		es.unregister <- client
	}()

	// Send events to client
	for {
		select {
		case <-r.Context().Done():
			return
		case message := <-client.events:
			fmt.Fprintf(w, "data: %s\n\n", message)
			flusher.Flush()
		}
	}
}

func (es *EventServer) broadcastMessage(message string) {
	es.broadcast <- message
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/hello" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method is not supported.", http.StatusNotFound)
		return
	}

	fmt.Fprintf(w, "Hello!")
}

func pingPongHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/ping" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method is not supported.", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"message": "pong"}`)
}

func main() {
	// Create and start event server
	eventServer := NewEventServer()
	go eventServer.run()

	// Set up routes
	http.HandleFunc("/hello", helloHandler)
	http.HandleFunc("/ping", pingPongHandler)
	http.HandleFunc("/events", eventServer.handleSSE)

	// Start a goroutine to send periodic test messages
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for t := range ticker.C {
			eventServer.broadcastMessage(fmt.Sprintf("Server time: %s", t.Format(time.RFC3339)))
		}
	}()

	fmt.Printf("Starting server at port 8080\n")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
