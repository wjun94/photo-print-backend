package utils

import (
	"log"
	"sync"

	"github.com/bwmarrin/snowflake"
)

var (
    once   sync.Once
    node   *snowflake.Node
)

// InitSnowflake 初始化雪花算法节点（机器ID，范围0-1023）
func InitSnowflake(machineID int64) {
    once.Do(func() {
        var err error
        node, err = snowflake.NewNode(machineID)
        if err != nil {
            log.Fatalf("初始化雪花算法失败: %v", err)
        }
        log.Printf("雪花算法初始化成功，机器ID: %d", machineID)
    })
}

// NextID 生成下一个唯一ID
func NextID() int64 {
    if node == nil {
        // 默认机器ID=1（生产环境应从配置读取）
        InitSnowflake(1)
    }
    return node.Generate().Int64()
}