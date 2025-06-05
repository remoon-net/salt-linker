package db

import "github.com/pocketbase/pocketbase/core"

type Endpoint struct {
	core.BaseModel
	User          string  `db:"user"`
	Device        string  `db:"device"`
	TransmitBytes float64 `db:"transmit_bytes"`
	Token         string  `db:"token"`
}

var _ core.Model = (*Endpoint)(nil)

func (Endpoint) TableName() string { return TableEndpoints }

const DeviceTable = "devices"
