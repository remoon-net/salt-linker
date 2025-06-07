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

		goods := try.To1(app.FindCollectionByNameOrId(db.TableGoods))
		goods.Fields.AddAt(getFieldIndex(goods, "price"),
			&core.SelectField{
				Name: "type", Id: ID("type"), System: true,
				Required: true, Presentable: true,
				Values: []string{"电子商品"}, MaxSelect: 1,
			},
		)
		goods.Fields.AddAt(getFieldIndex(goods, "poster")+1,
			&core.FileField{
				Name: "hookjs", Id: ID("hookjs"), System: true,
				Hidden:    true,
				MimeTypes: []string{"application/javascript", "text/javascript"}, MaxSelect: 1,
			},
		)
		try.To(app.Save(goods))

		return
	}, func(app core.App) error {
		return fmt.Errorf("fix orders rule rollback todo")
	})
}
