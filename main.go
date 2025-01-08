package main

import (
	"github.com/pocketbase/pocketbase"
	"github.com/shynome/err0/try"
	_ "remoon.net/salt-linker/migrations"
)

func main() {
	app := pocketbase.New()
	app.OnServe().BindFunc(initLinker)
	try.To(app.Start())
}
