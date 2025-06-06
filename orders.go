package main

import (
	"net/http"
	"slices"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/shynome/err0"
	"github.com/shynome/err0/try"
	"remoon.net/salt-linker/db"
)

func initOrders(e *core.ServeEvent) (err error) {
	defer err0.Then(&err, nil, nil)

	// 确保 item 的值是计算出来的
	e.App.OnRecordCreateRequest(db.TableOrderItems).BindFunc(computeItemValue)
	e.App.OnRecordUpdateRequest(db.TableOrderItems).BindFunc(computeItemValue)

	e.App.OnRecordCreateRequest(db.TableOrders).BindFunc(func(e *core.RecordRequestEvent) (err error) {
		defer err0.Then(&err, nil, nil)

		order := e.Record

		items := try.To1(e.App.FindRecordsByIds(db.TableOrderItems, order.GetStringSlice("items")))
		var value float64 = 0
		for _, item := range items {
			itemValue := item.GetFloat("value")
			value = value + itemValue
		}

		// 这里应该计算 coupons 的, 但没有coupons所以不用计算

		order.Set("status", types.JSONArray[db.OrderStatus]{db.OrderStatusWaitPay})
		order.Set("value", value)

		return e.Next()
	})
	e.App.OnRecordAfterCreateSuccess(db.TableOrders).BindFunc(func(e *core.RecordEvent) (err error) {
		defer err0.Then(&err, nil, nil)
		order := e.Record
		items := try.To1(e.App.FindRecordsByIds(db.TableOrderItems, order.GetStringSlice("items")))
		for _, item := range items {
			item.Set("order", order.Id)
			try.To(e.App.Save(item))
		}
		return e.Next()
	})
	e.App.OnRecordDeleteRequest(db.TableOrders).BindFunc(func(e *core.RecordRequestEvent) (err error) {
		defer err0.Then(&err, nil, nil)
		order := e.Record
		ss := order.GetStringSlice("status")
		ss = append(ss, string(db.OrderStatusClosed))
		order.Set("status", ss)
		try.To(e.App.Save(order))
		// 这里应该释放 coupons 的, 但没有coupons所以不用释放
		return e.JSON(http.StatusOK, order)
	})

	return e.Next()
}

func computeItemValue(e *core.RecordRequestEvent) (err error) {
	defer err0.Then(&err, nil, nil)
	item := e.Record
	goods := try.To1(e.App.FindRecordById(db.TableGoods, item.GetString("goods")))
	num := item.GetFloat("num")
	price := goods.GetFloat("price")
	value := num * price
	item.Set("value", value)
	return nil
}

var emptyJSONValues = []string{
	"null", `""`, "[]", "{}", "",
}

func IsEmptyJSON(s string) bool {
	s = strings.TrimSpace(s)
	return slices.Contains(emptyJSONValues, s)
}
