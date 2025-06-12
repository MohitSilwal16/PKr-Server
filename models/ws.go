package models

import (
	"sync"

	"github.com/gorilla/websocket"
)

type User struct {
	Username string
	Password string
}

type ConnManager struct {
	sync.RWMutex
	ConnPool map[string]*websocket.Conn
}

type WSMessage struct {
	User        User
	MessageType string // Error, NotifyToPunchResponse
	Message     any
}
