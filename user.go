package main

import (
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/shynome/err0"
	"github.com/shynome/err0/try"
	"remoon.net/salt-linker/db"
)

func initUser(app core.App) {
	// 用户不存在时请求 OTP 会失败, 在用户请求时创建该用户
	app.OnRecordRequestOTPRequest(db.TableUsers).BindFunc(func(e *core.RecordCreateOTPRequestEvent) (err error) {
		defer err0.Then(&err, nil, nil)
		if e.Record != nil {
			return e.Next()
		}
		var data struct {
			Email string `json:"email" form:"email"`
		}
		try.To(e.BindBody(&data))
		email := data.Email
		if email == "" {
			return apis.NewBadRequestError("missing email field", nil)
		}
		users := try.To1(app.FindCachedCollectionByNameOrId(db.TableUsers))
		user := core.NewRecord(users)
		user.SetEmail(email)
		user.SetRandomPassword()
		try.To(app.Save(user))
		e.Record = user
		return e.Next()
	})
}
