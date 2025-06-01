package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHelloHandler(t *testing.T) {
	// Create a request to pass to our handler
	req, err := http.NewRequest("GET", "/hello", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(helloHandler)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body
	expected := "Hello!"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestPingPongHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/ping", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(pingPongHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	if message, exists := response["message"]; !exists || message != "pong" {
		t.Errorf("handler returned unexpected message: got %v want %v",
			message, "pong")
	}
}

func TestSSEHandler(t *testing.T) {
	// Create a new event server
	es := NewEventServer()
	go es.run()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(es.handleSSE))
	defer server.Close()

	// Replace http with ws in URL
	url := fmt.Sprintf("http://%s", strings.TrimPrefix(server.URL, "http://"))

	// Make request to SSE endpoint
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// Check headers
	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("Expected Content-Type: text/event-stream, got %v", resp.Header.Get("Content-Type"))
	}

	// Test message broadcast
	testMessage := "Test message"
	go func() {
		time.Sleep(100 * time.Millisecond) // Give time for connection to establish
		es.broadcastMessage(testMessage)
	}()

	// Read response
	reader := bufio.NewReader(resp.Body)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	// Check if received message matches sent message
	expected := fmt.Sprintf("data: %s\n", testMessage)
	if line != expected {
		t.Errorf("Expected message %q, got %q", expected, line)
	}
}

func TestEventServerClientManagement(t *testing.T) {
	es := NewEventServer()
	go es.run()

	// Test client registration
	client := &Client{
		id:     "test-client",
		events: make(chan string, 10),
	}

	// Register client
	es.register <- client

	// Wait a bit for registration to complete
	time.Sleep(100 * time.Millisecond)

	// Verify client was registered
	es.mutex.RLock()
	if _, exists := es.clients[client.id]; !exists {
		t.Error("Client was not registered")
	}
	es.mutex.RUnlock()

	// Test message broadcast
	testMessage := "Test broadcast message"
	es.broadcastMessage(testMessage)

	// Verify client received message
	select {
	case msg := <-client.events:
		if msg != testMessage {
			t.Errorf("Expected message %q, got %q", testMessage, msg)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for message")
	}

	// Test client unregistration
	es.unregister <- client

	// Wait a bit for unregistration to complete
	time.Sleep(100 * time.Millisecond)

	// Verify client was unregistered
	es.mutex.RLock()
	if _, exists := es.clients[client.id]; exists {
		t.Error("Client was not unregistered")
	}
	es.mutex.RUnlock()
}

func TestMethodNotAllowed(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		method   string
		handler  http.HandlerFunc
		wantCode int
	}{
		{
			name:     "POST to hello endpoint",
			path:     "/hello",
			method:   "POST",
			handler:  helloHandler,
			wantCode: http.StatusNotFound,
		},
		{
			name:     "POST to ping endpoint",
			path:     "/ping",
			method:   "POST",
			handler:  pingPongHandler,
			wantCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(tt.handler)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.wantCode {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.wantCode)
			}
		})
	}
} 