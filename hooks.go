package main

import (
	"context"
	"fmt"
	"io"
	"slices"
	"sync"
	"time"

	"github.com/docker/go-units"
	"github.com/dop251/goja"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/jsvm"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/shynome/err0"
	"github.com/shynome/err0/try"
	"golang.org/x/sync/errgroup"
	"remoon.net/salt-linker/db"
	"remoon.net/salt-linker/hookjs"
)

func initHooks(se *core.ServeEvent) (err error) {
	defer err0.Then(&err, nil, nil)

	logger := se.App.Logger()

	pool := &sync.Pool{
		New: func() any {
			vm := goja.New()

			baseBind(vm)
			vm.Set("$app", se.App)

			vm.Set("GenLicense", GenLicense)
			vm.Set("bytes2str", bytes2str)

			try.To1(vm.RunProgram(hookjs.AlmondProg))
			return vm
		},
	}

	paidItems := make(chan *core.Record, 1024)
	se.App.OnRecordAfterUpdateSuccess(db.TableOrders).BindFunc(func(e *core.RecordEvent) error {
		ss := e.Record.GetStringSlice("status")
		paid := slices.Contains(ss, string(db.OrderStatusPaid))
		if !paid {
			return e.Next()
		}
		params := dbx.Params{
			"order": e.Record.Id,
		}
		items, err := e.App.FindRecordsByFilter(db.TableOrderItems, `order = {:order} && goods.hookjs != "" && executed = ""`, "", 0, 0, params)
		if err != nil {
			return err
		}
		for _, item := range items {
			paidItems <- item
		}
		return e.Next()
	})

	{ // 清理 fake executed
		q := dbx.HashExp{"executed": fakeExecuted}
		body := dbx.Params{"executed": ""}
		try.To1(se.App.DB().Update(db.TableOrderItems, body, q).Execute())
	}

	go func() {
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		se.App.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
			cancel()
			return e.Next()
		})
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case item := <-paidItems:
				if err := execHook(se.App, pool, item); err != nil {
					logger.Error("执行Goods hookjs失败了", "error", err, "id", item.Id)
				}
			case <-t.C:
				items, err := se.App.FindRecordsByFilter(db.TableOrderItems, `order.status:each ?= "已支付" && goods.hookjs != "" && executed = ""`, "order.updated", 10, 0)
				if err != nil {
					logger.Error("获取 order items 失败", "error", err)
					continue
				}
				if len(items) == 0 {
					continue
				}
				eg := new(errgroup.Group)
				for _, item := range items {
					eg.Go(func() error {
						return execHook(se.App, pool, item)
					})
				}
				if err := eg.Wait(); err != nil {
					var ids []string
					for _, items := range items {
						ids = append(ids, items.Id)
					}
					logger.Error("执行Goods hookjs失败了", "error", err, "ids", ids)
				}
			}
		}
	}()

	return se.Next()
}

var fakeExecuted, _ = types.ParseDateTime(time.Unix(0, 0))

func execHook(app core.App, pool *sync.Pool, item *core.Record) (err error) {
	defer err0.Then(&err, nil, nil)
	vm := pool.Get().(*goja.Runtime)
	defer pool.Put(vm)

	item = try.To1(app.FindRecordById(db.TableOrderItems, item.Id))
	if item.GetString("executed") != "" {
		return nil
	}

	item.Set("executed", fakeExecuted) // 添加fake executed, 避免 hookjs 中 update order 重复触发 hook
	try.To(app.Save(item))
	defer err0.Then(&err, nil, func() {
		item.Set("executed", "")
		try.To(app.Save(item))
	})

	goods := try.To1(app.FindRecordById(db.TableGoods, item.GetString("goods")))

	f := goods.BaseFilesPath() + "/" + goods.GetString("hookjs")

	fs := try.To1(app.NewFilesystem())
	r := try.To1(fs.GetReader(f))
	defer r.Close()
	s := string(try.To1(io.ReadAll(r)))

	s = hookjs.FixAlmondDefine(f, s)
	try.To1(vm.RunScript(f, s))

	exports := try.To1(vm.RunString(fmt.Sprintf(`requirejs("%s")`, f)))
	callback, ok := goja.AssertFunction(exports)
	if !ok {
		return fmt.Errorf("hookjs module.exports must be function")
	}
	try.To1(callback(goja.Undefined(), vm.ToValue(item)))

	item.Set("executed", types.NowDateTime())
	try.To(app.Save(item))

	return nil
}

func baseBind(vm *goja.Runtime) {
	vm.SetFieldNameMapper(jsvm.FieldMapper{})

	vm.Set("Record", func(call goja.ConstructorCall) *goja.Object {
		var instance *core.Record

		collection, ok := call.Argument(0).Export().(*core.Collection)
		if ok {
			instance = core.NewRecord(collection)
			data, ok := call.Argument(1).Export().(map[string]any)
			if ok {
				instance.Load(data)
			}
		} else {
			instance = &core.Record{}
		}

		instanceValue := vm.ToValue(instance).(*goja.Object)
		instanceValue.SetPrototype(call.This.Prototype())

		return instanceValue
	})
}

func bytes2str(b float64) string {
	return units.BytesSize(b)
}
