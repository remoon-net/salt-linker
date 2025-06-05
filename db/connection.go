package db

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

type Connection struct {
	core.BaseModel
	User          string         `db:"user"`
	Endpoint      string         `db:"endpoint"`
	TransmitBytes float64        `db:"transmit_bytes"`
	Disconnected  types.DateTime `db:"disconnected"`
	Metadata      types.JSONRaw  `db:"metadata"`
}

var _ core.Model = (*Connection)(nil)

func (Connection) TableName() string { return TableConnections }
