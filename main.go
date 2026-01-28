package main

import (
	"github.com/pocketbase/pocketbase"
	"github.com/shynome/err0/try"
	_ "remoon.net/salt-linker/migrations"
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
	initUser(app)

	try.To(app.Start())
}
