package db

import "github.com/pocketbase/pocketbase/core"

type User struct {
	core.BaseModel
	RemainingBytes float64 `db:"remaining_bytes"`
}

var _ core.Model = (*User)(nil)

func (User) TableName() string { return TableUsers }
