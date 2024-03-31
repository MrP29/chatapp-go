package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/websocket"
)

const (
	ROUTE_PREFIX      = "/chatapp"
	DEFAULT_NAME      = "Jane Smith"
	DEFAULT_EMAIL     = "janes@yorku.ca"
	DEFAULT_TOPIC     = "chat"
	REDIS_CHANNEL     = "messages"
	REDIS_ID_KEY      = "id"
	REDIS_MESSAGES_KEY = "messages"
)

type Message struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Date    string `json:"date"`
	Topic   string `json:"topic"`
	Content string `json:"content"`
}

func encodeMessages(messages []*Message) []byte {
	data, err := json.Marshal(messages)
	if err != nil {
		fmt.Println("Error encoding messages:", err)
		return []byte{}
	}
	return data
}

func sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	r.ParseForm()

	// Create a Message
	message := &Message{
		Name:    r.Form.Get("name"),
		Email:   r.Form.Get("email"),
		Topic:   r.Form.Get("topic"),
		Content: r.Form.Get("content"),
		Date:    time.Now().Format("01/02/2006 15:04:05"),
	}

	// Store the message in Redis
	conn, err := redis.Dial("tcp", "redis:6379")
	if err != nil {
		http.Error(w, "Error connecting to Redis", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	messageID, err := redis.Int(conn.Do("INCR", REDIS_ID_KEY))
	if err != nil {
		http.Error(w, "Error generating message ID", http.StatusInternalServerError)
		return
	}

	message.ID = messageID

	_, err = conn.Do("RPUSH", REDIS_MESSAGES_KEY, encodeMessages([]*Message{message}))
	if err != nil {
		http.Error(w, "Error storing message in Redis", http.StatusInternalServerError)
		return
	}

	// Broadcast the message to all clients
	conn.Do("PUBLISH", REDIS_CHANNEL, encodeMessages([]*Message{message}))
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Error upgrading to WebSocket", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// Open a connection to Redis
	rConn, err := redis.Dial("tcp", "redis:6379")
	if err != nil {
		fmt.Println("Error connecting to Redis:", err)
		return
	}
	defer rConn.Close()

	// Subscribe to Redis channel
	psc := redis.PubSubConn{Conn: rConn}
	psc.Subscribe(REDIS_CHANNEL)

	// Listen for messages from Redis and send them to WebSocket clients
	for {
		switch v := psc.Receive().(type) {
		case redis.Message:
			conn.WriteMessage(websocket.TextMessage, v.Data)
		case redis.Subscription:
			fmt.Printf("Subscribed to channel %s\n", v.Channel)
		case error:
			fmt.Println("Error:", v)
			return
		}
	}
}

func main() {
	http.HandleFunc(ROUTE_PREFIX+"/send", sendMessageHandler)
	http.HandleFunc(ROUTE_PREFIX+"/websocket", websocketHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("Server running on port", port)
	http.ListenAndServe(":"+port, nil)
}
