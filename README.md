# Simple Go Web Server

A minimal Go web server that exposes a `/hello` endpoint.

## Features
- Simple HTTP server running on port 8080
- `/hello` endpoint that returns a greeting
- Basic error handling for invalid paths and methods

## Running the Server
```bash
go run main.go
```

Then visit `http://localhost:8080/hello` in your browser or use curl:
```bash
curl http://localhost:8080/hello
``` 