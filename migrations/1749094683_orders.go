package migrations

import (
	"fmt"

	"github.com/docker/go-units"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/shynome/err0"
	"github.com/shynome/err0/try"
	"remoon.net/salt-linker/db"
)

func init() {
	migrations.Register(func(app core.App) (err error) {
		defer err0.Then(&err, nil, nil)

		users := try.To1(app.FindCollectionByNameOrId("users"))

		goods := core.NewBaseCollection("goods", ID("goods"))
		goods.ViewRule = types.Pointer("")
		goods.ListRule = types.Pointer("hide = false")
		goods.Fields.Add(
			&core.TextField{
				Name: "name", Id: ID("name"), System: true,
				Required: true, Presentable: true,
			},
			&core.NumberField{
				Name: "order", Id: ID("order"), System: true,
				OnlyInt: true,
			},
			&core.BoolField{
				Name: "hide", Id: ID("hide"), System: true,
			},
			&core.BoolField{
				Name: "disabled", Id: ID("disabled"), System: true,
			},
			&core.NumberField{
				Name: "price", Id: ID("price"), System: true,
				Required: true,
				OnlyInt:  true, Min: types.Pointer[float64](0),
			},
			&core.FileField{
				Name: "poster", Id: ID("poster"), System: true,
			},
			&core.EditorField{
				Name: "desc", Id: ID("desc"), System: true,
			},
		)
		addUpdatedFields(&goods.Fields)
		try.To(app.Save(goods))

		coupon := core.NewBaseCollection("coupon", ID("coupon"))
		coupon.ViewRule = types.Pointer("")
		coupon.Fields.Add(
			&core.TextField{
				Name: "name", Id: ID("name"), System: true,
				Required: true, Presentable: true,
			},
			&core.NumberField{
				Name: "value", Id: ID("value"), System: true,
				Required: true,
				OnlyInt:  true, Min: types.Pointer[float64](0),
			},
			&core.DateField{
				Name: "expires_at", Id: ID("expires_at"), System: true,
				Required: true,
			},
			&core.RelationField{
				Name: "goods", Id: ID("goods"), System: true,
				CollectionId: goods.Id, MaxSelect: 9999,
			},
		)
		addUpdatedFields(&coupon.Fields)
		try.To(app.Save(coupon))

		var orders *core.Collection

		couponIssued := core.NewBaseCollection("coupon_issued", ID("coupon_issued"))
		couponIssued.ListRule = types.Pointer("@request.auth.id = user")
		couponIssued.ViewRule = types.Pointer("@request.auth.id = user")
		couponIssued.Fields.Add(
			&core.RelationField{
				Name: "user", Id: ID("user"), System: true,
				Required:     true,
				CollectionId: users.Id, MaxSelect: 1, CascadeDelete: true,
			},
			&core.RelationField{
				Name: "coupon", Id: ID("coupon"), System: true,
				Required: true, Presentable: true,
				CollectionId: coupon.Id, MaxSelect: 1,
			},
			&core.TextField{
				Name: "order", Id: ID("order_placeholder"),
				Required: true,
			},
		)
		defer err0.Then(&err, func() {
			couponIssued.Fields.RemoveById(ID("order_placeholder"))
			couponIssued.Fields.AddAt(getFieldIndex(couponIssued, "coupon")+1,
				&core.RelationField{
					Name: "order", Id: ID("order"), System: true,
					Required:     true,
					CollectionId: orders.Id, MaxSelect: 1,
				},
			)
			try.To(app.Save(couponIssued))
		}, nil)
		addUpdatedFields(&couponIssued.Fields)
		try.To(app.Save(couponIssued))

		items := core.NewBaseCollection("order_items", ID("order_items"))
		items.ListRule = types.Pointer("@request.auth.id = user")
		items.ViewRule = types.Pointer("@request.auth.id = user")
		items.CreateRule = types.Pointer(`@request.auth.id = user && @request.body.order:isset = false`)
		items.UpdateRule = types.Pointer(`@request.auth.id = user && order = "" && @request.body.goods:isset = false`)
		items.DeleteRule = types.Pointer(`@request.auth.id = user && order = ""`)
		items.Fields.Add(
			&core.RelationField{
				Name: "user", Id: ID("user"), System: true,
				Required:     true,
				CollectionId: users.Id, MaxSelect: 1,
			},
			&core.TextField{
				Name: "order", Id: ID("order_placeholder"),
			},
			&core.RelationField{
				Name: "goods", Id: ID("goods"), System: true,
				Required: true, Presentable: true,
				CollectionId: goods.Id, MaxSelect: 1,
			},
			&core.NumberField{
				Name: "num", Id: ID("num"), System: true,
				Required: true, Presentable: true,
				OnlyInt: true, Min: types.Pointer[float64](1),
			},
			&core.NumberField{
				Name: "value", Id: ID("value"), System: true,
				OnlyInt: true, Min: types.Pointer[float64](0),
			},
		)
		defer err0.Then(&err, func() {
			items.Fields.RemoveById(ID("order_placeholder"))
			items.Fields.AddAt(getFieldIndex(items, "user")+1,
				&core.RelationField{
					Name: "order", Id: ID("order"), System: true,
					CollectionId: orders.Id, MaxSelect: 1,
				},
			)
			try.To(app.Save(items))
		}, nil)
		addUpdatedFields(&items.Fields)
		try.To(app.Save(items))

		express := core.NewBaseCollection("express", ID("express"))
		express.ListRule = types.Pointer("@request.auth.id = user")
		express.ViewRule = types.Pointer("@request.auth.id = user")
		express.Fields.Add(
			&core.RelationField{
				Name: "user", Id: ID("user"), System: true,
				Required:     true,
				CollectionId: users.Id, MaxSelect: 1,
			},
			&core.TextField{
				Name: "order", Id: ID("order_placeholder"),
				Required: true,
			},
			&core.RelationField{
				Name: "items", Id: ID("items"), System: true,
				CollectionId: items.Id, MaxSelect: 99999,
			},
			&core.JSONField{
				Name: "value", Id: ID("value"), System: true,
				Required: true,
			},
			&core.TextField{
				Name: "remark", Id: ID("remark"), System: true,
			},
		)
		defer err0.Then(&err, func() {
			express.Fields.RemoveById(ID("order_placeholder"))
			express.Fields.AddAt(getFieldIndex(express, "user")+1,
				&core.RelationField{
					Name: "order", Id: ID("order"), System: true,
					Required:     true,
					CollectionId: orders.Id, MaxSelect: 1,
				},
			)
			try.To(app.Save(express))
		}, nil)
		addUpdatedFields(&express.Fields)
		try.To(app.Save(express))

		orders = core.NewBaseCollection("orders", ID("orders"))
		orders.ListRule = types.Pointer("@request.auth.id = user")
		orders.ViewRule = types.Pointer("@request.auth.id = user")
		orders.CreateRule = types.Pointer("@request.auth.id = user")
		orders.UpdateRule = types.Pointer("@request.auth.id = user")
		orders.DeleteRule = types.Pointer("@request.auth.id = user")
		orders.Fields.Add(
			&core.RelationField{
				Name: "user", Id: ID("user"), System: true,
				Required:     true,
				CollectionId: users.Id, MaxSelect: 1,
			},
			&core.SelectField{
				Name: "status", Id: ID("status"), System: true,
				MaxSelect: 6, Values: []string{
					string(db.OrderStatusWaitPay),
					string(db.OrderStatusPaid),
					string(db.OrderStatusCount),
					string(db.OrderStatusMaking),
					string(db.OrderStatusSended),
					string(db.OrderStatusClosed),
				},
			},
			&core.NumberField{
				Name: "value", Id: ID("value"), System: true,
				Required: true,
				Min:      types.Pointer[float64](0),
			},
			&core.RelationField{
				Name: "items", Id: ID("items"), System: true,
				Required:     true,
				CollectionId: items.Id, MaxSelect: 99999,
			},
			&core.RelationField{
				Name: "coupons", Id: ID("coupons"), System: true,
				Required:     true,
				CollectionId: couponIssued.Id, MaxSelect: 99999,
			},
			&core.JSONField{
				Name: "address", Id: ID("address"), System: true,
				MaxSize: 100 * units.KiB,
			},
			&core.RelationField{
				Name: "express", Id: ID("express"), System: true,
				CollectionId: express.Id, MaxSelect: 99999,
			},
		)
		try.To(app.Save(orders))

		return nil
	}, func(app core.App) error {
		return fmt.Errorf("orders no rollback")
	})
}
