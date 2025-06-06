package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
	"github.com/shynome/err0"
	"github.com/shynome/err0/try"
	"remoon.net/salt-linker/db"
)

func init() {
	migrations.Register(func(app core.App) (err error) {
		defer err0.Then(&err, nil, nil)

		orders := try.To1(app.FindCollectionByNameOrId(db.TableOrders))
		orders.Fields.AddAt(getFieldIndex(orders, "payment_created_info")+1,
			&core.JSONField{
				Name: "payment_callbacked_info", Id: ID("payment_callbacked_info"), System: true,
				Hidden: true,
			},
		)
		try.To(app.Save(orders))

		return
	}, func(app core.App) error {
		return fmt.Errorf("fix orders rule rollback todo")
	})
}
