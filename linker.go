package main

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/yamux"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/store"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/shynome/err0"
	"github.com/shynome/err0/try"
	"github.com/shynome/websocket"
	"remoon.net/salt-linker/db"
)

var linkers = &store.Store[string, *WrapperProxy]{}

func initLinker(se *core.ServeEvent) (err error) {
	defer err0.Then(&err, nil, nil)
	app := se.App

	{ //将上次退出时未设置为断开的链接设置为断开
		d := dbx.Params{"disconnected": types.NowDateTime()}
		w := dbx.HashExp{"disconnected": ""}
		q := app.DB().Update(db.TableConnections, d, w)
		try.To1(q.Execute())
	}

	et := try.To1(app.FindCollectionByNameOrId(db.TableEndpoints))
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

	app.OnRecordAfterUpdateSuccess(db.TableConnections).BindFunc(func(e *core.RecordEvent) error {
		disconnected := e.Record.GetDateTime("disconnected")
		if disconnected.IsZero() {
			return e.Next()
		}
		uid := e.Record.GetString("user")
		tx := e.Record.GetFloat("transmit_bytes")
		if err := e.App.RunInTransaction(func(txApp core.App) (err error) {
			user := try.To1(txApp.FindRecordById(db.TableUsers, uid))
			rb := user.GetFloat("remaining_bytes")
			rb -= tx
			user.Set("remaining_bytes", rb)
			return txApp.Save(user)
		}); err != nil {
			return err
		}
		return e.Next()
	})
	app.OnRecordAfterUpdateSuccess(db.TableConnections).BindFunc(func(e *core.RecordEvent) error {
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
			ep := try.To1(txApp.FindRecordById(db.TableEndpoints, eid))
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
	se.Router.GET("/api/salt/whip/{ep}", SaltLinkerServe)
	se.Router.GET("/api/salt/link/{ep}/{token}", SaltLinker)

	app.OnRecordAfterDeleteSuccess(db.DeviceTable).BindFunc(func(e *core.RecordEvent) error {
		k := e.Record.GetString("endpoint")
		wp, ok := linkers.GetOk(k)
		if ok {
			wp.Cancel()
		}
		return nil
	})

	return se.Next()
}

func SaltLinker(e *core.RequestEvent) (err error) {
	defer err0.Then(&err, nil, nil)
	r := e.Request
	w := e.Response
	socket := try.To1(websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
		Subprotocols:   []string{"link"},
	}))
	id := r.PathValue("ep")
	if id == "" {
		return socket.Close(4000+http.StatusBadRequest, "endpoint is required")
	}
	app := e.App
	var ep db.Endpoint
	try.To(app.ModelQuery(&ep).Where(dbx.HashExp{"id": id}).One(&ep))
	token := r.PathValue("token")
	if ep.Token != token {
		return socket.Close(4000+http.StatusUnauthorized, "token is incorrect")
	}
	if ep.Device == "" {
		return socket.Close(4000+http.StatusPreconditionFailed, "this endpoint is unbind device")
	}
	ctx := r.Context()
	ctx, cacnel := context.WithCancel(ctx)
	defer cacnel()
	conn := websocket.NetConn(ctx, socket, websocket.MessageBinary)
	rwc := &RWCounter{ReadWriteCloser: conn}
	sess := try.To1(yamux.Client(rwc, nil))
	defer sess.Close()

	target, _ := url.Parse("http://yamux.proxy/")
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return sess.Open()
		},
	}

	if _, ok := linkers.GetOk(id); ok {
		return socket.Close(4000+http.StatusLocked, "device is already connected")
	}
	wp := &WrapperProxy{ReverseProxy: proxy, Cancel: cacnel}
	linkers.Set(id, wp)
	defer linkers.Remove(id)

	{
		s := http.NewServeMux()
		s.HandleFunc("PATCH /direct-link", func(w http.ResponseWriter, r *http.Request) {
			link := r.FormValue("link")
			if link == "" {
				http.Error(w, "link is lost", http.StatusBadRequest)
				return
			}
			if len(link) > 120 {
				http.Error(w, "link is too large", http.StatusBadRequest)
				return
			}
			if _, err := url.Parse(link); err != nil {
				http.Error(w, "link parse failed"+err.Error(), http.StatusBadRequest)
				return
			}
			wp.DirectLink = link
			w.WriteHeader(http.StatusNoContent)
		})
		go http.Serve(sess, s)
	}

	metadata := try.To1(json.Marshal(Metadata{
		Method:     r.Method,
		RemoteAddr: e.RealIP(),
		RequestURI: r.RequestURI,
		Header:     r.Header,
	}))

	c := try.To1(app.FindCollectionByNameOrId(db.TableConnections))
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
	id := r.PathValue("ep")
	if id == "" {
		return apis.NewUnauthorizedError("unkown endpoint", nil)
	}
	proxy, ok := linkers.GetOk(id)
	if !ok {
		return apis.NewApiError(http.StatusServiceUnavailable, "device is offline", nil)
	}
	// 只允许 WebScoket 连接
	if upgrade := r.Header.Get("Upgrade"); !strings.EqualFold(upgrade, "websocket") {
		e.Response.Header().Set("Upgrade", "websocket")
		return apis.NewApiError(http.StatusUpgradeRequired, "device is online (only allow websocket connection)", nil)
	}
	if proxy.DirectLink != "" {
		conn, err := websocket.Accept(e.Response, r, &websocket.AcceptOptions{
			OriginPatterns: []string{"*"},
			Subprotocols:   []string{"wgortc"},
		})
		if err != nil {
			return apis.NewInternalServerError("redirect websocket accept failed", err)
		}
		return conn.Close(3307, proxy.DirectLink)
	}
	r.Body = NotRereadable(r.Body)
	proxy.ServeHTTP(e.Response, r)
	return nil
}

type WrapperProxy struct {
	*httputil.ReverseProxy
	Cancel     context.CancelFunc
	DirectLink string
}

type Metadata struct {
	Method     string
	RemoteAddr string
	RequestURI string
	Header     http.Header
}

// 双向计费, 因为 Read 也是出流量 client->server->linker, Write 则是 linker->server->client
type RWCounter struct {
	io.ReadWriteCloser
	count atomic.Int64
}

func (rwc *RWCounter) Count() float64 {
	c := rwc.count.Load()
	return float64(c)
}

var _ io.ReadWriter = (*RWCounter)(nil)

func (rwc *RWCounter) Write(p []byte) (n int, err error) {
	n, err = rwc.ReadWriteCloser.Write(p)
	if err == nil {
		rwc.count.Add(int64(n))
	}
	return n, err
}

func (rwc *RWCounter) Read(p []byte) (n int, err error) {
	n, err = rwc.ReadWriteCloser.Read(p)
	if err == nil {
		rwc.count.Add(int64(n))
	}
	return n, err
}

type NotRereadableBody struct {
	io.ReadCloser
	end error
}

func NotRereadable(body io.ReadCloser) *NotRereadableBody {
	return &NotRereadableBody{
		ReadCloser: body,
	}
}

func (b *NotRereadableBody) Read(p []byte) (n int, err error) {
	if b.end != nil {
		return 0, b.end
	}
	n, err = b.ReadCloser.Read(p)
	if err == io.EOF {
		b.end = err
	}
	return
}
