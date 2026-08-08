package main

import (
	"encoding/json"
	"testing"
	"time"
)

// 用内存 channel 模拟两个 client，验证广播和上下线提示。
func TestHubBroadcast(t *testing.T) {
	h := newHub()
	go h.run()

	c1 := &client{hub: h, send: make(chan []byte, 8), user: "甲"}
	c2 := &client{hub: h, send: make(chan []byte, 8), user: "乙"}
	h.register <- c1
	h.register <- c2
	time.Sleep(20 * time.Millisecond)

	// 注册时两人都会收到对方的进场提示，先把 c2 的提示排空，只留真正要验证的。
	drain := func(ch chan []byte) {
		for {
			select {
			case <-ch:
			default:
				return
			}
		}
	}
	drain(c2.send)

	// 甲发一条，乙应该收到，甲自己也会收到（网页端自己也会显示自己的消息）。
	msg, _ := json.Marshal(Message{User: "甲", Text: "你好"})
	h.broadcast <- msg
	time.Sleep(20 * time.Millisecond)

	select {
	case got := <-c2.send:
		var m Message
		_ = json.Unmarshal(got, &m)
		if m.Text != "你好" || m.User != "甲" {
			t.Fatalf("乙收到内容不对: %+v", m)
		}
	default:
		t.Fatal("乙没收到广播")
	}

	// 甲下线，应该产生一条系统提示广播。
	h.unregister <- c1
	time.Sleep(20 * time.Millisecond)
	select {
	case got := <-c2.send:
		var m Message
		_ = json.Unmarshal(got, &m)
		if m.User != "系统" {
			t.Fatalf("下线提示发送者应为系统, 得到 %q", m.User)
		}
	default:
		t.Fatal("乙没收到下线提示")
	}
}
