# Simple Go Web Server

A minimal Go web server that exposes a `/hello` endpoint and Server-Sent Events support.

## Features
- Simple HTTP server running on port 8080
- `/hello` endpoint that returns a greeting
- `/ping` endpoint that returns a JSON response
- `/events` endpoint for Server-Sent Events (SSE)
- Basic error handling for invalid paths and methods

## Running the Server
```bash
go run main.go
```

## Endpoints

### Hello Endpoint
```bash
curl http://localhost:8080/hello
```

### Ping Endpoint
```bash
curl http://localhost:8080/ping
```

### SSE Events Endpoint
To connect to the SSE endpoint and receive real-time updates:

```bash
curl -N http://localhost:8080/events
```

Or use JavaScript in a web browser:
```javascript
const eventSource = new EventSource('http://localhost:8080/events');
eventSource.onmessage = function(event) {
    console.log('Received:', event.data);
};
```

The server will send a timestamp message every 5 seconds to all connected clients. 