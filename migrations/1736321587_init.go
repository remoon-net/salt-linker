package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/shynome/err0"
	"github.com/shynome/err0/try"
)

func init() {
	migrations.Register(func(app core.App) (err error) {
		defer err0.Then(&err, nil, nil)

		users := try.To1(app.FindCollectionByNameOrId("users"))
		users.ListRule = types.Pointer("id = @request.auth.id")
		users.ViewRule = types.Pointer("id = @request.auth.id")
		users.CreateRule = types.Pointer("")
		users.UpdateRule = types.Pointer(`id = @request.auth.id 
&& @request.body.remaining_bytes:isset = false`)
		users.DeleteRule = nil // 不允许删除用户
		users.Fields.Add(&core.NumberField{
			Id:      "__remaining_bytes__",
			Name:    "remaining_bytes",
			OnlyInt: true,
		})
		try.To(app.Save(users))

		endpoints := core.NewBaseCollection("endpoints", "__endpoints__")
		endpoints.ListRule = types.Pointer("user = @request.auth.id")
		endpoints.ViewRule = types.Pointer("user = @request.auth.id")
		endpoints.Fields.Add(
			&core.RelationField{
				Id: "__user__", Name: "user", System: true,
				CascadeDelete: true,
				Required:      true,
				CollectionId:  users.Id,
				MaxSelect:     1,
			},
			&core.NumberField{
				Id: "__transmit_bytes__", Name: "transmit_bytes", System: true,
				OnlyInt: true,
				Min:     types.Pointer[float64](0),
			},
			&core.TextField{
				Id: "__token__", Name: "token", System: true,
				Required: true,
			},
			&core.AutodateField{
				Id: "__created__", Name: "created", System: true,
				OnCreate: true,
			},
			&core.AutodateField{
				Id: "__updated__", Name: "updated", System: true,
				OnCreate: true,
				OnUpdate: true,
			},
		)
		try.To(app.Save(endpoints))

		devices := core.NewBaseCollection("devices", "__devices__")
		devices.ListRule = types.Pointer("user = @request.auth.id")
		devices.ViewRule = types.Pointer("user = @request.auth.id")
		devices.CreateRule = types.Pointer("user = @request.auth.id")
		devices.UpdateRule = types.Pointer(`user = @request.auth.id
&& @request.body.endpoint:isset = false`)
		devices.DeleteRule = types.Pointer("user = @request.auth.id")
		devices.Fields.Add(
			&core.RelationField{
				Id: "__user__", Name: "user", System: true,
				CascadeDelete: true,
				Required:      true,
				CollectionId:  users.Id,
				MaxSelect:     1,
			},
			&core.TextField{
				Id: "__name__", Name: "name", System: true,
				Required:    true,
				Max:         200,
				Presentable: true,
			},
			&core.RelationField{
				Id: "__endpoint__", Name: "endpoint", System: true,
				CollectionId: endpoints.Id,
				MaxSelect:    1,
			},
			&core.AutodateField{
				Id: "__created__", Name: "created", System: true,
				OnCreate: true,
			},
			&core.AutodateField{
				Id: "__updated__", Name: "updated", System: true,
				OnCreate: true,
				OnUpdate: true,
			},
		)
		try.To(app.Save(devices))

		endpoints.Fields.AddAt(2, &core.RelationField{
			Id: "__device__", Name: "device", System: true,
			CascadeDelete: false,
			CollectionId:  devices.Id,
			MaxSelect:     1,
		})
		try.To(app.Save(endpoints))

		connections := core.NewBaseCollection("connections", "__connections__")
		connections.ListRule = types.Pointer("user = @request.auth.id")
		connections.ViewRule = types.Pointer("user = @request.auth.id")
		connections.Fields.Add(
			&core.RelationField{
				Id: "__user__", Name: "user", System: true,
				CascadeDelete: true,
				Required:      true,
				CollectionId:  users.Id,
				MaxSelect:     1,
			},
			&core.RelationField{
				Id: "__endpoint__", Name: "endpoint", System: true,
				CascadeDelete: true,
				Required:      true,
				CollectionId:  endpoints.Id,
				MaxSelect:     1,
			},
			&core.NumberField{
				Id: "__transmit_bytes__", Name: "transmit_bytes", System: true,
				OnlyInt: true,
				Min:     types.Pointer[float64](0),
			},
			&core.DateField{
				Id: "__disconnected__", Name: "disconnected", System: true,
			},
			&core.JSONField{
				Id: "__metadata__", Name: "metadata", System: true,
				Hidden: true,
			},
			&core.AutodateField{
				Id: "__created__", Name: "created", System: true,
				OnCreate: true,
			},
			&core.AutodateField{
				Id: "__updated__", Name: "updated", System: true,
				OnCreate: true,
				OnUpdate: true,
			},
		)
		try.To(app.Save(connections))

		return
	}, func(app core.App) error {
		return fmt.Errorf("init db rollback todo")
	})
}
