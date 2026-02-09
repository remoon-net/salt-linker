package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/shynome/err0/try"
	"remoon.net/salt-linker/db"
	_ "remoon.net/salt-linker/db/migrations"
)

var args struct {
	PSC         string
	Money1Bytes string
}

func main() {
	app := pocketbase.New()

	{
		flags := app.RootCmd.PersistentFlags()
		flags.StringVar(&args.PSC, "psc", "", "支付中心的接口地址, WebSocket 链接")
		flags.StringVar(&args.Money1Bytes, "m1b", "1GB", "一块钱能买多少流量")
	}

	app.OnServe().BindFunc(initLinker)
	app.OnServe().BindFunc(initPSC)
	app.OnServe().BindFunc(bindInstallerFiles)
	initUser(app)

	try.To(app.Start())
}

func bindInstallerFiles(e *core.ServeEvent) error {

	e.Router.GET("/api/collections/installer_files/platforms", func(e *core.RequestEvent) (err error) {
		h := e.Response.Header()
		h.Set("Cache-Control", "public, max-age=60")

		platforms := []string{"guide", "server", "windows", "android", "linux"}
		sf := filepath.Join(e.App.DataDir(), "supported_platforms.json")
		if b, err := os.ReadFile(sf); err == nil {
			_ = json.Unmarshal(b, &platforms)
		}

		files := map[string]any{}
		for _, p := range platforms {
			files[p] = getIFiles(e.App, p)
		}
		return e.JSON(http.StatusOK, files)
	})
	return e.Next()
}

func getIFiles(app core.App, platform string) *core.Record {
	q := `platform = {:platform} && download != ''`
	p := dbx.Params{"platform": platform}
	ifiles, err := app.FindRecordsByFilter(db.TableInstallerFiles, q, "-code,-created", 1, 0, p)
	if err != nil {
		return nil
	}
	if len(ifiles) == 0 {
		return nil
	}
	return ifiles[0]
}
