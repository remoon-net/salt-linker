package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/docker/go-units"
	"github.com/hashicorp/yamux"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/shynome/err0"
	"github.com/shynome/err0/try"
	"github.com/shynome/websocket"
	"remoon.net/salt-linker/db"
	"resty.dev/v3"
)

var PSCSess atomic.Pointer[yamux.Session]
var phc = resty.New().SetBaseURL("http://payment.service")

func init() {
	phc.SetTransport(&http.Transport{
		DisableKeepAlives: true,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			sess := PSCSess.Load()
			if sess == nil {
				return nil, apis.NewApiError(http.StatusPreconditionFailed, "支付中心尚未准备好", nil)
			}
			return sess.Open()
		},
	})
	phc.SetRedirectPolicy(resty.NoRedirectPolicy())
	phc.SetRetryCount(2).SetRetryWaitTime(time.Second)
}

func initPSC(e *core.ServeEvent) (err error) {
	defer err0.Then(&err, nil, nil)
	logger := e.App.Logger()

	if args.PSC == "" {
		return e.Next()
	}

	m1b := try.To1(units.FromHumanSize(args.Money1Bytes))

	pr := router.NewRouter(func(w http.ResponseWriter, r *http.Request) (*core.RequestEvent, router.EventCleanupFunc) {
		event := new(core.RequestEvent)
		event.Response = w
		event.Request = r
		event.App = e.App

		return event, nil
	})
	pr.Any("/callback", func(e *core.RequestEvent) (err error) {
		defer err0.Then(&err, nil, nil)
		var payment Payment
		try.To(e.BindBody(&payment))
		u := try.To1(url.Parse(payment.Link))
		id := strings.TrimPrefix(u.Fragment, "order-")
		if id == "" {
			return apis.NewBadRequestError("缺少id", nil)
		}
		var cbRaw types.JSONRaw = try.To1(json.Marshal(payment))
		err = retry.Do(func() error {
			return e.App.RunInTransaction(func(tx core.App) (err error) {
				order := try.To1(tx.FindRecordById(db.TableOrders, id))
				if cb := order.GetString("payment_callbacked_info"); IsEmptyJSON(cb) {
					order.Set("payment_callbacked_info", cbRaw)
				} else {
					e.App.Logger().Warn("本应只会回调一次怎么回调了两次", "cb", payment, "order", id)
				}
				ss := order.GetStringSlice("status")
				closed := slices.Contains(ss, string(db.OrderStatusClosed))
				paid := slices.Contains(payment.Status, db.PaymentStatusPaid)
				if closed { // closed order 就单纯存一下回调数据
					try.To(tx.Save(order))
					return nil
				}
				if paid {
					user := try.To1(tx.FindRecordById(db.TableUsers, order.GetString("user")))
					b := user.GetFloat("remaining_bytes")
					g := user.GetFloat("bytes")
					b = b + g
					user.Set("remaining_bytes", b)
					try.To(tx.Save(user))

					ss = append(ss, string(db.OrderStatusPaid))
					ss = slices.DeleteFunc(ss, func(s string) bool {
						return s == string(db.OrderStatusWaitPay)
					})
				}
				ss = append(ss, string(db.OrderStatusClosed))
				order.Set("status", ss)
				try.To(tx.Save(order))
				return nil
			})
		},
			retry.Attempts(3),
		)
		try.To(err)
		return e.NoContent(http.StatusNoContent)
	})
	srv := try.To1(pr.BuildMux())

	go retry.Do(func() (err error) {
		defer err0.Then(&err, nil, func() {
			logger.Error("PSC failed", "error", err)
		})

		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		socket, _ := try.To2(websocket.Dial(ctx, args.PSC, nil))
		conn := websocket.NetConn(ctx, socket, websocket.MessageBinary)

		sess := try.To1(yamux.Client(conn, nil))

		PSCSess.Store(sess)
		defer PSCSess.Store(nil)

		return http.Serve(sess, srv)
	},
		retry.Attempts(0),
		retry.MaxDelay(20*time.Second),
	)

	e.App.OnRecordCreateRequest(db.TableOrders).BindFunc(func(e *core.RecordRequestEvent) (err error) {
		order := e.Record
		num := order.GetFloat("value")
		g := num / 100 * float64(m1b)
		order.Set("bytes", g)
		return e.Next()
	})
	e.App.OnRecordUpdateRequest(db.TableOrders).BindFunc(func(e *core.RecordRequestEvent) (err error) {
		defer err0.Then(&err, nil, nil)

		info := try.To1(e.RequestInfo())
		if len(info.Body) > 0 && !info.Auth.IsSuperuser() {
			return apis.NewBadRequestError("只有管理员可以更改信息", nil)
		}

		order := e.Record
		if plink := order.Get("payment_link"); plink == "" {
			g := order.GetFloat("bytes")
			s := units.HumanSize(g)
			resp, err := phc.R().
				SetBody(map[string]any{
					"name":  fmt.Sprintf("购买流量包: %s", s),
					"value": order.GetInt("value"),
					"link":  orderLink(e.App, order.Id),
				}).
				Post("/payments")
			try.To(err)
			if code := resp.StatusCode(); code != http.StatusSeeOther {
				z := resp.String()
				return apis.NewInternalServerError("创建 Payment 失败了", z)
			}

			var payment Payment
			try.To(json.NewDecoder(resp.Body).Decode(&payment))

			plink := resp.Header().Get("Location")
			order.Set("payment_link", plink)
			order.Set("status", []string{string(db.OrderStatusWaitPay)})
			var pcinfo types.JSONRaw
			pcinfo = try.To1(json.Marshal(payment))
			order.Set("payment_created_info", pcinfo)
		}

		return e.Next()
	})
	e.App.OnRecordDeleteRequest(db.TableOrders).Bind(&hook.Handler[*core.RecordRequestEvent]{
		Func: func(e *core.RecordRequestEvent) error {
			go func() {
				var payment Payment
				pcinfo := e.Record.GetString("payment_created_info")
				if err := json.Unmarshal([]byte(pcinfo), &payment); err != nil {
					return
				}
				if payment.ID == "" {
					return
				}
				phc.R().Delete("/payments/" + payment.ID)
			}()
			return e.Next()
		},
		Priority: -1,
	})

	return e.Next()
}

func orderLink(app core.App, id string) string {
	au, _ := url.Parse(app.Settings().Meta.AppURL)
	au = au.JoinPath("/user/orders/")
	// q := au.Query()
	// q.Set("id", id)
	// au.RawQuery = q.Encode()
	au.Fragment = "order-" + id
	r := au.String()
	return r
}

type Payment struct {
	ID          string             `json:"id"`
	Application string             `json:"application"`
	Name        string             `json:"name"`
	Value       int                `json:"value"`
	Link        string             `json:"link"`
	Callbacked  string             `json:"callbacked"`
	Status      []db.PaymentStatus `json:"status"`
	Details     []string           `json:"details"`
	Updated     string             `json:"updated"`
	Created     string             `json:"created"`
}
