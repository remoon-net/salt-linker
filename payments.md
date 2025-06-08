# 如何对接支付系统

要求服务端必须是 golang 写的, 所有接口的返回状态码成功时都是 303

接口运行在 [websocket](https://github.com/coder/websocket) 上的 [yamux.Session](https://pkg.go.dev/github.com/hashicorp/yamux#Server) 上

服务端示例实现, 服务端会在适当时候时候调用客户端的 [`/callback`](./psc.go#L63)

```go
package main

import (
	"context"
	"net"
	"net/http"

	"github.com/hashicorp/yamux"
	"github.com/shynome/websocket"
)

func main() {
	// payment service routes
	ps := http.NewServeMux()
	redirect2pay := func(w http.ResponseWriter) {
		h := w.Header()
		h.Set("Location", "http://payment.service/frontend/pay/page/?id={id}")
		w.WriteHeader(http.StatusSeeOther)
	}
  // 创建支付订单
	ps.HandleFunc("POST /payments", func(w http.ResponseWriter, r *http.Request) {
		redirect2pay(w)
	})
  // 获取支付订单
	ps.HandleFunc("GET /payments/{id}", func(w http.ResponseWriter, r *http.Request) {
		redirect2pay(w)
	})
  // 关闭支付订单
	ps.HandleFunc("DELETE /payments/{id}", func(w http.ResponseWriter, r *http.Request) {
		redirect2pay(w)
	})

	s := http.NewServeMux()
	s.HandleFunc("/pay/ws/link", func(w http.ResponseWriter, r *http.Request) {
		socket, _ := websocket.Accept(w, r, nil)
		ctx := r.Context()
		conn := websocket.NetConn(ctx, socket, websocket.MessageBinary)
		sess, _ := yamux.Server(conn, nil)
		hc := &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return sess.Open()
				},
				DisableKeepAlives: true,
			},
		}
		_ = hc // 你应该在某个地方保存hc, 以便后续调用客户端的 `/callback`
		http.Serve(sess, ps)
	})
	http.ListenAndServe(":8091", s)
}

```
