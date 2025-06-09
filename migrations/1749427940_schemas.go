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

		schemas := core.NewBaseCollection(db.TableSchemas, ID(db.TableSchemas))
		schemas.ViewRule = types.Pointer("")
		schemas.Fields.Add(
			&core.TextField{
				Name: "name", Id: ID("name"), System: true,
				Required: true, Presentable: true,
			},
			&core.JSONField{
				Name: "schema", Id: ID("schema"), System: true,
				Required: true,
			},
		)
		addUpdatedFields(&schemas.Fields)
		try.To(app.Save(schemas))

		goods := try.To1(app.FindCollectionByNameOrId(db.TableGoods))
		goods.Fields.AddAt(getFieldIndex(goods, "price")+1,
			&core.RelationField{
				Name: "schema", Id: ID("schema"), System: true,
				CollectionId: schemas.Id, MaxSelect: 1,
			},
		)
		try.To(app.Save(goods))

		return nil
	}, func(app core.App) error {
		return fmt.Errorf("schemas no rollback")
	})
}
