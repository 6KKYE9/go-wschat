# go-wschat

TCP 聊天室的升级版：换成 WebSocket，再配一个网页，打开浏览器就能聊，不用自己写客户端。后端还是单个 hub 单循环管所有连接，只是传输层从裸 TCP 换成了 WebSocket。

## 跑起来

```powershell
go mod tidy   # 第一次会拉 gorilla/websocket
go run .
```

然后浏览器开 http://127.0.0.1:9001 ，多开几个标签页就是几个人在聊。

换端口：

```powershell
go run . -addr :9002
```

## 怎么做的

- **握手**：`gorilla/websocket` 的 upgrader 把 HTTP 连接升级成 WebSocket。练手阶段 `CheckOrigin` 直接放行，真上线要按域名白名单收一下。
- **hub 单循环**：注册、注销、广播全走 channel 丢给 `run()` 里的一个 goroutine 处理，避免多个 goroutine 同时写一个连接把消息写串。
- **每连接两 goroutine**：`readPump` 一直读，`writePump` 一直从 `send` channel 往外写。读到的消息带上昵称后丢进 `broadcast`。
- **广播防卡**：某个客户端发送缓冲满了（比如网页卡死没读），直接摘掉它，不连累其他人。
- **系统提示**：有人进/出聊天室，hub 会发一条 `User:"系统"` 的消息，前端用灰色斜体显示。

## 和 TCP 版的区别

| | go-tcpchat | go-wschat |
|---|---|---|
| 传输 | 裸 TCP | WebSocket |
| 客户端 | telnet / 自带 client | 浏览器网页 |
| 消息格式 | 每行一条文本 | JSON `{user, text}` |
| 依赖 | 零依赖 | gorilla/websocket |

## 没做的事

- 没鉴权、没房间（所有人一个大厅）、没历史消息
- 单进程单 hub，要多机得加 Redis 之类的中间层

## 测试

```powershell
go test ./...
```

hub 的广播和上下线提示用内存 channel 模拟两个 client 测了，不依赖真实网络。
