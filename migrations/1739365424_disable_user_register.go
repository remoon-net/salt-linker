package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
	"github.com/shynome/err0"
	"github.com/shynome/err0/try"
)

func init() {
	migrations.Register(func(app core.App) (err error) {
		defer err0.Then(&err, nil, nil)

		users := try.To1(app.FindCollectionByNameOrId("users"))
		users.CreateRule = nil // 禁止用户自行注册, 如果要开发注册, 请自行修改Rule为 ""
		try.To(app.Save(users))

		return
	}, func(app core.App) error {
		return fmt.Errorf("disable user register rollback todo")
	})
}
