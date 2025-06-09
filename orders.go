package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/santhosh-tekuri/jsonschema/v6"
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

		// verify address
		gIds := []string{}
		for _, item := range items {
			gIds = append(gIds, item.GetString("goods"))
		}
		goods := try.To1(e.App.FindRecordsByIds(db.TableGoods, gIds))

		schemas := map[string]string{}
		for _, g := range goods {
			sid := g.GetString("schema")
			if sid == "" {
				continue
			}
			schema := try.To1(e.App.FindRecordById(db.TableSchemas, sid))
			schemas[g.Id] = schema.GetString("schema")
		}

		addrs := map[string]json.RawMessage{}
		try.To(json.Unmarshal([]byte(order.GetString("address")), &addrs))

		compiler := jsonschema.NewCompiler()
		for _, item := range items {
			gid := item.GetString("goods")
			schemaRaw, ok := schemas[gid]
			if !ok {
				continue
			}

			var g *core.Record
			if idx := slices.IndexFunc(goods, func(item *core.Record) bool { return item.Id == gid }); idx != -1 {
				g = goods[idx]
			}
			gDisplay := gid
			if g != nil {
				gDisplay = g.GetString("name")
			}

			addrRaw, ok := addrs[item.Id]
			if !ok {
				msg := fmt.Sprintf("商品(%s)要求地址输入, 但没有对应项的地址输入", gDisplay)
				return apis.NewBadRequestError(msg, nil)
			}

			res := try.To1(jsonschema.UnmarshalJSON(strings.NewReader(schemaRaw)))
			try.To(compiler.AddResource(gid, res))
			schema := try.To1(compiler.Compile(gid))
			var v any
			try.To(json.Unmarshal(addrRaw, &v))
			if err := schema.Validate(v); err != nil {
				msg := fmt.Sprintf("未通过商品(%s)的地址输入验证. 错误原因: %s", gDisplay, err.Error())
				return apis.NewBadRequestError(msg, err)
			}
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
		info := try.To1(e.RequestInfo())
		if info.Auth.IsSuperuser() {
			return e.Next()
		}
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

	// 添加 item_num 数量限制, 因为有些商品的 hookjs 只支持单个商品, 数量超过一个时也只处理一个
	limit := goods.GetInt("item_num_limit")
	num := item.GetInt("num")
	if limit > 0 && num > limit {
		num = limit
		item.Set("num", num)
	}

	price := goods.GetInt("price")
	value := num * price
	item.Set("value", value)

	return e.Next()
}

var emptyJSONValues = []string{
	"null", `""`, "[]", "{}", "",
}

func IsEmptyJSON(s string) bool {
	s = strings.TrimSpace(s)
	return slices.Contains(emptyJSONValues, s)
}
