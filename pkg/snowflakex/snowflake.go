package snowflakex

import (
	"github.com/bwmarrin/snowflake"
	"sync"
)

var (
	sf   *snowflake.Node
	once sync.Once
)

// InitSnowflake 初始化雪花节点
// nodeID 范围：0 ~ 1023（因为 snowflake 默认 10 位节点 ID）
func InitSnowflake(nodeID int64) {
	once.Do(func() {
		var err error
		sf, err = snowflake.NewNode(nodeID)
		if err != nil {
			panic("初始化雪花ID失败: " + err.Error())
		}
	})
}

// NextID 生成全局唯一 int64 ID
func NextID() int64 {
	return sf.Generate().Int64()
}
