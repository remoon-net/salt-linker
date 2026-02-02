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

		iFiles := core.NewBaseCollection(db.TableInstallerFiles, ID(db.TableInstallerFiles))
		iFiles.ListRule = types.Pointer("")
		iFiles.ViewRule = types.Pointer("")
		iFiles.CreateRule = nil
		iFiles.UpdateRule = nil
		iFiles.DeleteRule = nil
		iFiles.Fields.Add(
			&core.TextField{
				Name: "filename", Id: ID("filename"), System: true,
			},
			&core.TextField{
				Name: "platform", Id: ID("platform"), System: true,
			},
			&core.TextField{
				Name: "version", Id: ID("version"), System: true,
			},
			&core.NumberField{
				Name: "code", Id: ID("code"), System: true,
				OnlyInt: true,
			},
			&core.URLField{
				Name: "download", Id: ID("download"), System: true,
			},
		)
		addUpdatedFields(&iFiles.Fields)
		try.To(app.Save(iFiles))

		return nil
	}, func(app core.App) error {
		return fmt.Errorf("orders no rollback")
	})
}
