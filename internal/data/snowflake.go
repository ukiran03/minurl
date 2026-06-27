package data

import (
	"github.com/bwmarrin/snowflake"
)

func NewSnowflakeID(sfnid int64) snowflake.ID {
	node, _ := snowflake.NewNode(sfnid)
	return node.Generate()
}
