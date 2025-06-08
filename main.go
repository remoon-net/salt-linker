package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/shynome/err0/try"
	_ "remoon.net/salt-linker/migrations"
)

var args struct {
	PSC        string
	LicenseKey []byte
}

func main() {
	app := pocketbase.New()

	{
		flags := app.RootCmd.PersistentFlags()
		flags.StringVar(&args.PSC, "psc", "", "支付中心的接口地址, WebSocket 链接")
		flags.BytesBase64Var(&args.LicenseKey, "license-key", nil, "生成 license 的 key, base64编码")
	}

	app.OnServe().BindFunc(initLinker)
	app.OnServe().BindFunc(initOrders)
	app.OnServe().BindFunc(initPSC)
	app.OnServe().BindFunc(initHooks)

	var publicDir string
	app.RootCmd.PersistentFlags().StringVar(
		&publicDir,
		"publicDir",
		defaultPublicDir(),
		"the directory to serve static files",
	)

	// static route to serves files from the provided public dir
	// (if publicDir exists and the route path is not already defined)
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("/{path...}", apis.Static(os.DirFS(publicDir), true))

		return e.Next()
	})

	try.To(app.Start())
}

func defaultPublicDir() string {
	if strings.HasPrefix(os.Args[0], os.TempDir()) {
		// most likely ran with go run
		return "./pb_public"
	}

	return filepath.Join(os.Args[0], "../pb_public")
}
