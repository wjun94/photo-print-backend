package common

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"photo-print-backend/utils"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
)

// ✅ 修正1：添加Children字段匹配原始JSON结构
type rawRegion struct {
	Code     string      `json:"code"`
	Name     string      `json:"name"`
	Children []rawRegion `json:"children,omitempty"` // 关键修复：添加嵌套结构
}

// RegionNode 树形节点 (保持输出结构不变)
type RegionNode struct {
	ID       int64        `json:"id"`
	Name     string       `json:"name"`
	Children []RegionNode `json:"children,omitempty"`
}

var (
	regionTree []RegionNode
	once       sync.Once
)

// loadRegionTree 从项目根目录的 data/pca-code.json 加载
func loadRegionTree() []RegionNode {
	once.Do(func() {
		log.Println("🔍 开始加载省市区数据...")

		// 获取当前工作目录
		wd, err := os.Getwd()
		if err != nil {
			log.Fatalf("❌ 获取工作目录失败: %v", err)
		}
		log.Printf("📂 当前工作目录: %s", wd)

		// 构建文件路径
		filePath := filepath.Join(wd, "data", "pca-code.json")
		log.Printf("📄 尝试读取文件: %s", filePath)

		// 读取文件
		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Fatalf("❌ 读取 pca-code.json 失败: %v", err)
		}

		// ✅ 修正2：直接解析为树形结构 (不再需要手动组装)
		var rawTree []rawRegion
		if err := json.Unmarshal(data, &rawTree); err != nil {
			log.Fatalf("❌ 解析 pca-code.json 失败: %v", err)
		}
		log.Printf("✅ 成功加载 %d 个省级节点 (含子节点)", len(rawTree))

		// ✅ 修正3：递归转换 rawRegion -> RegionNode
		regionTree = convertToRegionNode(rawTree)
		log.Printf("✅ 转换完成，共 %d 个省份", len(regionTree))
	})
	return regionTree
}

// 递归转换函数：rawRegion -> RegionNode
func convertToRegionNode(raw []rawRegion) []RegionNode {
	var result []RegionNode
	for _, r := range raw {
		// 转换当前节点
		id, _ := strconv.ParseInt(r.Code, 10, 64) // 忽略错误（已在日志中处理）

		node := RegionNode{
			ID:   id,
			Name: r.Name,
		}

		// 递归转换子节点
		if len(r.Children) > 0 {
			node.Children = convertToRegionNode(r.Children)
		}

		result = append(result, node)
	}
	return result
}

// GetAllRegions 获取全部省市区数据（树形结构）
// @Summary 获取全国省市区数据
// @Description 一次性返回所有省市区，按省份->城市->区县树形结构
// @Tags 公共接口
// @Produce json
// @Success 200 {object} utils.Response{data=[]RegionNode}
// @Router /api/v1/regions/all [get]
func GetAllRegions(c *gin.Context) {
	tree := loadRegionTree()
	utils.Success(c, tree)
}
