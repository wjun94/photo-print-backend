package utils

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type rawRegion struct {
	Code     string      `json:"code"`
	Name     string      `json:"name"`
	Children []rawRegion `json:"children,omitempty"`
}

var (
	regionNameMap map[string]string
	regionOnce    sync.Once // 改为 regionOnce，避免与 snowflake 冲突
)

// loadRegionTree 从项目根目录的 data/pca-code.json 加载，并递归构建 id->name 映射
func loadRegionTree() {
	regionOnce.Do(func() {
		log.Println("🔍 开始加载省市区数据...")

		wd, err := os.Getwd()
		if err != nil {
			log.Fatalf("❌ 获取工作目录失败: %v", err)
		}
		log.Printf("📂 当前工作目录: %s", wd)

		filePath := filepath.Join(wd, "data", "pca-code.json")
		log.Printf("📄 尝试读取文件: %s", filePath)

		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Fatalf("❌ 读取 pca-code.json 失败: %v", err)
		}

		var rawTree []rawRegion
		if err := json.Unmarshal(data, &rawTree); err != nil {
			log.Fatalf("❌ 解析 pca-code.json 失败: %v", err)
		}
		log.Printf("✅ 成功加载 %d 个省级节点", len(rawTree))

		regionNameMap = make(map[string]string)

		// 递归收集所有节点的 code 和 name
		var collect func(nodes []rawRegion)
		collect = func(nodes []rawRegion) {
			for _, node := range nodes {
				if node.Code != "" && node.Code != "0" {
					regionNameMap[node.Code] = node.Name
				}
				if len(node.Children) > 0 {
					collect(node.Children)
				}
			}
		}
		collect(rawTree)

		log.Printf("✅ 省市区数据加载完成，共 %d 条记录", len(regionNameMap))
	})
}

// GetRegionName 根据 ID 获取名称，如果不存在返回空字符串
func GetRegionName(id string) string {
	loadRegionTree()
	return regionNameMap[id]
}
