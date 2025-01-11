package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/hashicorp/yamux"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/shynome/err0"
	"github.com/shynome/err0/try"
	"remoon.net/salt-linker/db"
)

func initLinker(se *core.ServeEvent) (err error) {
	defer err0.Then(&err, nil, nil)
	app := se.App

	{ //将上次退出时未设置为断开的链接设置为断开
		d := dbx.Params{"disconnected": types.NowDateTime()}
		w := dbx.HashExp{"disconnected": ""}
		q := app.DB().Update(db.ConnectionTable, d, w)
		try.To1(q.Execute())
	}

	et := try.To1(app.FindCollectionByNameOrId(db.EndpointTable))
	app.OnRecordAfterCreateSuccess("devices").BindFunc(func(e *core.RecordEvent) error {
		app := e.App
		ep := core.NewRecord(et)
		ep.Load(map[string]any{
			"user":   e.Record.GetString("user"),
			"device": e.Record.Id,
			"token":  uuid.NewString(),
		})
		if err := app.Save(ep); err != nil {
			return err
		}
		e.Record.Set("endpoint", ep.Id)
		if err := app.Save(e.Record); err != nil {
			return err
		}
		return e.Next()
	})

	app.OnRecordAfterUpdateSuccess(db.ConnectionTable).BindFunc(func(e *core.RecordEvent) error {
		disconnected := e.Record.GetDateTime("disconnected")
		if disconnected.IsZero() {
			return e.Next()
		}
		uid := e.Record.GetString("user")
		tx := e.Record.GetFloat("transmit_bytes")
		if err := e.App.RunInTransaction(func(txApp core.App) (err error) {
			user := try.To1(txApp.FindRecordById(db.UserTable, uid))
			rb := user.GetFloat("remaining_bytes")
			rb -= tx
			user.Set("remaining_bytes", rb)
			return txApp.Save(user)
		}); err != nil {
			return err
		}
		return e.Next()
	})
	app.OnRecordAfterUpdateSuccess(db.ConnectionTable).BindFunc(func(e *core.RecordEvent) error {
		disconnected := e.Record.GetDateTime("disconnected")
		if disconnected.IsZero() {
			return e.Next()
		}
		eid := e.Record.GetString("endpoint")
		if eid == "" {
			return e.Next()
		}
		tx := e.Record.GetFloat("transmit_bytes")
		if err := e.App.RunInTransaction(func(txApp core.App) (err error) {
			ep := try.To1(txApp.FindRecordById(db.EndpointTable, eid))
			count := ep.GetFloat("transmit_bytes")
			count += tx
			ep.Set("transmit_bytes", count)
			return txApp.Save(ep)
		}); err != nil {
			return err
		}
		return e.Next()
	})

	// se.Router.GET("/link/status", SaltLinkerStatus)
	se.Router.Any("/api/salt-link", SaltLinkerServe)
	se.Router.Any("/api/salt-link/{token}", SaltLinker)
	return se.Next()
}

func SaltLinker(e *core.RequestEvent) (err error) {
	defer err0.Then(&err, nil, nil)
	r := e.Request
	sp := r.Header.Get("Sec-Websocket-Protocol")
	if !strings.HasPrefix(sp, "link") {
		return apis.NewBadRequestError("subprotocol is bad", nil)
	}
	protocols := strings.Split(sp, ", ")
	if len(protocols) != 2 {
		return apis.NewBadRequestError("unkown endpoint", nil)
	}
	id := protocols[1]
	if id == "" {
		return apis.NewBadRequestError("endpoint is required", nil)
	}
	app := e.App
	var ep db.Endpoint
	try.To(app.ModelQuery(&ep).Where(dbx.HashExp{"id": id}).One(&ep))
	token := r.PathValue("token")
	if ep.Token != token {
		return apis.NewBadRequestError("token is incorrect", nil)
	}
	w := e.Response
	socket := try.To1(websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
		Subprotocols:   []string{"link"},
	}))
	ctx := r.Context()
	conn := websocket.NetConn(ctx, socket, websocket.MessageBinary)
	rwc := &WriteCounter{ReadWriteCloser: conn}
	sess := try.To1(yamux.Client(rwc, nil))
	defer sess.Close()

	target, _ := url.Parse("http://yamux.proxy/")
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return sess.Open()
		},
	}

	store := app.Store()
	k := fmt.Sprintf("salt-linker-%s", id)
	if _, ok := store.GetOk(k); ok {
		return socket.Close(websocket.StatusAbnormalClosure, "device is already connected")
	}

	metadata := try.To1(json.Marshal(Metadata{
		Method:     r.Method,
		RemoteAddr: r.RemoteAddr,
		RequestURI: r.RequestURI,
		Header:     r.Header,
	}))

	c := try.To1(app.FindCollectionByNameOrId(db.ConnectionTable))
	connection := core.NewRecord(c)
	connection.Load(map[string]any{
		"user":     ep.User,
		"endpoint": ep.Id,
		"metadata": metadata,
	})
	try.To(app.Save(connection))

	defer func() {
		connection.Set("transmit_bytes", rwc.Count())
		connection.Set("disconnected", types.NowDateTime())
		app.Save(connection)
	}()

	store.Set(k, proxy)
	defer store.Remove(k)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
				connection.Set("transmit_bytes", rwc.Count())
				app.Save(connection)
			}
		}
	}()

	<-sess.CloseChan()
	return nil
}

func SaltLinkerServe(e *core.RequestEvent) error {
	r := e.Request
	if r.Method == http.MethodGet {
		return apis.NewApiError(http.StatusMethodNotAllowed, "只允许非GET请求通过, 避免浏览器注入", nil)
	}
	id, _, _ := r.BasicAuth()
	if id == "" {
		return apis.NewUnauthorizedError("unkown endpoint", nil)
	}
	k := fmt.Sprintf("salt-linker-%s", id)
	p, ok := e.App.Store().GetOk(k)
	if !ok {
		return apis.NewApiError(http.StatusServiceUnavailable, "device is offline", nil)
	}
	proxy, ok := p.(*httputil.ReverseProxy)
	if !ok {
		return apis.NewInternalServerError("there should have a reverse proxy server", nil)
	}
	proxy.ServeHTTP(e.Response, r)
	return nil
}

type Metadata struct {
	Method     string
	RemoteAddr string
	RequestURI string
	Header     http.Header
}

type WriteCounter struct {
	io.ReadWriteCloser
	count atomic.Int64
}

func (rwc *WriteCounter) Count() float64 {
	c := rwc.count.Load()
	return float64(c)
}

var _ io.Writer = (*WriteCounter)(nil)

func (rwc *WriteCounter) Write(p []byte) (n int, err error) {
	n, err = rwc.ReadWriteCloser.Write(p)
	if err == nil {
		rwc.count.Add(int64(n))
	}
	return n, err
}
