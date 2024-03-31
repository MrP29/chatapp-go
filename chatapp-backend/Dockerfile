# Start from a minimal Golang base image
FROM golang:1.17-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the source code into the container
COPY . .

Run go get github.com/gomodule/redigo/redis
Run go get github.com/gorilla/websocket

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o chatapp .

# Start from a minimal Alpine Linux base image
FROM alpine:latest

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/chatapp .

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./chatapp"]
