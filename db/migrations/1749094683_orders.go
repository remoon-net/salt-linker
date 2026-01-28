package migrations

import (
	"fmt"

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

		var orders *core.Collection

		orders = core.NewBaseCollection(db.TableOrders, ID(db.TableOrders))
		orders.ListRule = types.Pointer("@request.auth.id = user")
		orders.ViewRule = types.Pointer("@request.auth.id = user")
		orders.CreateRule = types.Pointer("@request.auth.id = user")
		orders.UpdateRule = types.Pointer("@request.auth.id = user") // 仅用以触发创建支付订单, 实际上是无法修改的
		orders.DeleteRule = types.Pointer(`@request.auth.id = user && status:each ?!= "已关闭"`)
		orders.Fields.Add(
			&core.RelationField{
				Name: "user", Id: ID("user"), System: true,
				Required:     true,
				CollectionId: users.Id, MaxSelect: 1,
			},
			&core.SelectField{
				Name: "status", Id: ID("status"), System: true,
				MaxSelect: 3, Values: []string{
					string(db.OrderStatusWaitPay),
					string(db.OrderStatusPaid),
					string(db.OrderStatusClosed),
				},
			},
			&core.NumberField{
				Name: "value", Id: ID("value"), System: true,
				Required: true,
				Min:      types.Pointer[float64](0), OnlyInt: true,
			},
			&core.NumberField{
				Name: "bytes", Id: ID("bytes"), System: true,
				Required: false,
				OnlyInt:  true,
			},
			&core.URLField{
				Name: "payment_link", Id: ID("payment_link"), System: true,
			},
			&core.JSONField{
				Name: "payment_created_info", Id: ID("payment_created_info"), System: true,
				Hidden: true,
			},
			&core.JSONField{
				Name: "payment_callbacked_info", Id: ID("payment_callbacked_info"), System: true,
				Hidden: true,
			},
		)
		addUpdatedFields(&orders.Fields)
		try.To(app.Save(orders))

		return nil
	}, func(app core.App) error {
		return fmt.Errorf("orders no rollback")
	})
}
