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

// 不允许使用已被使用了的订单项或优惠券, 不能使用其他人的订单项和券
const t1749137729OrdersCreateRule = `@request.auth.id = user 
&& items.order = "" 
&& items.user = @request.auth.id
&& coupons.order = ""
&& coupons.user = @request.auth.id`

// 已关闭的订单不再允许进行操作
const t1749137729OrdersDeleteRule = `@request.auth.id = user 
&& status:each ?!= "已关闭"`

func init() {
	migrations.Register(func(app core.App) (err error) {
		defer err0.Then(&err, nil, nil)

		orders := try.To1(app.FindCollectionByNameOrId(db.TableOrders))
		orders.CreateRule = types.Pointer(t1749137729OrdersCreateRule)
		orders.UpdateRule = types.Pointer(`@request.auth.id = user`) // 被 hook 控制, 只是用以触发请求支付
		orders.DeleteRule = types.Pointer(t1749137729OrdersDeleteRule)
		orders.Fields.AddAt(getFieldIndex(orders, "value")+1,
			&core.URLField{
				Name: "payment_link", Id: ID("payment_link"), System: true,
			},
			&core.JSONField{
				Name: "payment_created_info", Id: ID("payment_created_info"), System: true,
				Hidden: true,
			},
		)
		addUpdatedFields(&orders.Fields) //初始化的时候忘记加了
		try.To(app.Save(orders))

		return
	}, func(app core.App) error {
		return fmt.Errorf("fix orders rule rollback todo")
	})
}
