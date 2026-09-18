package handlers

import (
	"github.com/gin-gonic/gin"
	"name/internal/application/response"
	"name/internal/domain/hanzi"
	"name/internal/domain/name"
	"name/internal/infrastructure/logger"
	"go.uber.org/zap"
)

// StandardCharGroup 标准起名用字分组
type StandardCharGroup struct {
	Radical string   `json:"radical"`
	Name    string   `json:"name"`
	Meaning string   `json:"meaning"`
	Chars   []string `json:"chars"`
}

// CharacterHandler 汉字/偏旁数据服务
type CharacterHandler struct {
	nameDB *name.NameDB // 精选名来源（含自学习合并）
}

// NewCharacterHandler 创建汉字/偏旁数据服务
func NewCharacterHandler(nameDB *name.NameDB) *CharacterHandler {
	return &CharacterHandler{nameDB: nameDB}
}

// GetCharGroups 获取所有偏旁分组
// @Summary 获取偏旁分组
// @Description 获取按偏旁分组的标准起名用字
// @Tags 汉字数据
// @Accept json
// @Produce json
// @Success 200 {object} response.Response "成功"
// @Router /v1/characters/groups [get]
func (h *CharacterHandler) GetCharGroups(c *gin.Context) {
	groups, err := h.loadCharGroups()
	if err != nil {
		logger.Error("加载偏旁分组失败", zap.Error(err))
		response.ErrorJSON(c, 500, "加载偏旁分组失败")
		return
	}
	response.SuccessJSON(c, groups)
}

// 偏旁分组数据已并入 namer.json 顶层 charGroups（单一文件真源），由 hanzi 包
// 在启动加载时解析。此处直接从内存取用，不再读盘 standard_chars.json。
func (h *CharacterHandler) loadCharGroups() ([]StandardCharGroup, error) {
	groups := hanzi.GetNamerGroups()
	result := make([]StandardCharGroup, 0, len(groups))
	for _, g := range groups {
		result = append(result, StandardCharGroup{
			Radical: g.Radical,
			Name:    g.Name,
			Meaning: g.Meaning,
			Chars:   g.Chars,
		})
	}
	return result, nil
}
