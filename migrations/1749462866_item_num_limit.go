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

		goods := try.To1(app.FindCollectionByNameOrId(db.TableGoods))
		goods.Fields.AddAt(getFieldIndex(goods, "price")+1,
			&core.NumberField{
				Name: "item_num_limit", Id: ID("item_num_limit"), System: true,
				OnlyInt: true, Min: types.Pointer[float64](0),
			},
		)
		try.To(app.Save(goods))

		return
	}, func(app core.App) error {
		return fmt.Errorf("item_num_limit no rollback")
	})
}
