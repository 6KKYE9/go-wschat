package main

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	// 本地练手不校验 Origin，真上线要按域名白名单放开。
	CheckOrigin: func(r *http.Request) bool { return true },
}

func serveWS(h *hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := &client{hub: h, conn: conn, send: make(chan []byte, 256)}
	h.register <- c
	go c.writePump()
	go c.readPump()
}
