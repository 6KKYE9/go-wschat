package main

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

// Message 是客户端之间传递的消息结构。
type Message struct {
	User string `json:"user"`
	Text string `json:"text"`
}

// client 代表一个连上来的浏览器标签页。
type client struct {
	hub  *hub
	conn *websocket.Conn
	send chan []byte
	user string
}

// hub 维护所有在线连接，统一做广播。和 tcpchat 一样用单循环避免并发写。
type hub struct {
	mu      sync.Mutex
	clients map[*client]bool
	// 注册 / 注销 / 广播 都用 channel 丢给 run() 里的单循环处理。
	register   chan *client
	unregister chan *client
	broadcast  chan []byte
}

func newHub() *hub {
	return &hub{
		clients:    make(map[*client]bool),
		register:   make(chan *client),
		unregister: make(chan *client),
		broadcast:  make(chan []byte),
	}
}

func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = true
			h.mu.Unlock()
			h.broadcastText(c.user + " 进入了聊天室")
		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()
			h.broadcastText(c.user + " 离开了聊天室")
		case msg := <-h.broadcast:
			h.mu.Lock()
			for c := range h.clients {
				select {
				case c.send <- msg:
				default:
					// 某个客户端发送缓冲满了，直接摘掉，免得卡住整个广播。
					delete(h.clients, c)
					close(c.send)
				}
			}
			h.mu.Unlock()
		}
	}
}

// broadcastText 给所有在线 client 发一条系统提示。
// 只在 run() 的循环里调用（持有 mu 或正在处理 case），直接推给每个 client 的 send，
// 不绕 broadcast channel，避免和 run 自身死锁。
func (h *hub) broadcastText(text string) {
	b, _ := json.Marshal(Message{User: "系统", Text: text})
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		select {
		case c.send <- b:
		default:
			delete(h.clients, c)
			close(c.send)
		}
	}
}

func (c *client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(1 << 16) // 单条消息上限 64KB
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var m Message
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		if m.User != "" {
			c.user = m.User
		}
		out, _ := json.Marshal(Message{User: c.user, Text: m.Text})
		c.hub.broadcast <- out
	}
}

func (c *client) writePump() {
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
	c.conn.Close()
}
