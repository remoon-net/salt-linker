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

		items := try.To1(app.FindCollectionByNameOrId(db.TableOrderItems))
		items.Fields.AddAt(getFieldIndex(items, "created"),
			// 这个字段不加任何保护, 如果用户错误的设置此项, 那他就是不想要自动发货
			&core.DateField{
				Name: "executed", Id: ID("executed"), System: true,
			},
		)
		addUpdatedFields(&items.Fields) //初始化的时候忘记加了
		try.To(app.Save(items))

		return
	}, func(app core.App) error {
		return fmt.Errorf("fix orders rule rollback todo")
	})
}
