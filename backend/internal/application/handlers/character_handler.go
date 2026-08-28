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

// CuratedName 精选候选名（API 返回格式，与 domain 层解耦）
type CuratedName struct {
	Name        string   `json:"name"`
	Pinyin      string   `json:"pinyin"`
	Gender      string   `json:"gender"`
	Source      string   `json:"source"`
	Meaning     string   `json:"meaning"`
	Wuxing      string   `json:"wuxing"`
	YinyunScore float64  `json:"yinyun_score"`
	Styles      []string `json:"styles,omitempty"`
	Tags        []string `json:"tags,omitempty"`
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

// GetRadicalChars 获取指定偏旁下的汉字
// @Summary 按偏旁获取汉字
// @Description 按偏旁获取该偏旁下的所有标准起名用字
// @Tags 汉字数据
// @Accept json
// @Produce json
// @Param radical query string true "偏旁部首"
// @Success 200 {object} response.Response "成功"
// @Router /v1/characters/radical [get]
func (h *CharacterHandler) GetRadicalChars(c *gin.Context) {
	radical := c.Query("radical")
	if radical == "" {
		response.ErrorJSON(c, 400, "偏旁参数不能为空")
		return
	}

	groups, err := h.loadCharGroups()
	if err != nil {
		logger.Error("加载偏旁分组失败", zap.Error(err))
		response.ErrorJSON(c, 500, "加载偏旁分组失败")
		return
	}

	for _, g := range groups {
		if g.Radical == radical {
			response.SuccessJSON(c, g)
			return
		}
	}

	response.ErrorJSON(c, 404, "未找到该偏旁")
}

// GetCuratedNames 获取精选候选名库
// @Summary 获取精选候选名库
// @Description 获取精选候选名列表，支持按性别和风格筛选
// @Tags 汉字数据
// @Accept json
// @Produce json
// @Param gender query string false "性别筛选 (male/female)"
// @Param style query string false "风格筛选"
// @Success 200 {object} response.Response "成功"
// @Router /v1/characters/curated-names [get]
func (h *CharacterHandler) GetCuratedNames(c *gin.Context) {
	gender := c.Query("gender")
	style := c.Query("style")

	names, err := h.loadCuratedNames()
	if err != nil {
		logger.Error("加载候选名库失败", zap.Error(err))
		response.ErrorJSON(c, 500, "加载候选名库失败")
		return
	}

	var filtered []CuratedName
	for _, n := range names {
		if gender != "" && n.Gender != gender && n.Gender != "通用" {
			continue
		}
		if style != "" {
			hasStyle := false
			for _, s := range n.Styles {
				if s == style {
					hasStyle = true
					break
				}
			}
			if !hasStyle {
				continue
			}
		}
		filtered = append(filtered, n)
	}

	if filtered == nil {
		filtered = []CuratedName{}
	}

	response.SuccessJSON(c, filtered)
}

// GetStyles 获取所有风格标签
// @Summary 获取风格标签列表
// @Description 获取所有可用的候选名风格标签
// @Tags 汉字数据
// @Accept json
// @Produce json
// @Success 200 {object} response.Response "成功"
// @Router /v1/characters/styles [get]
func (h *CharacterHandler) GetStyles(c *gin.Context) {
	names, err := h.loadCuratedNames()
	if err != nil {
		logger.Error("加载候选名库失败", zap.Error(err))
		response.ErrorJSON(c, 500, "加载候选名库失败")
		return
	}

	styleSet := make(map[string]bool)
	for _, n := range names {
		for _, s := range n.Styles {
			styleSet[s] = true
		}
	}

	var styles []string
	for s := range styleSet {
		styles = append(styles, s)
	}

	if styles == nil {
		styles = []string{}
	}

	response.SuccessJSON(c, styles)
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

func (h *CharacterHandler) loadCuratedNames() ([]CuratedName, error) {
	if h.nameDB == nil {
		return nil, nil
	}
	domainNames := h.nameDB.GetCuratedNames("")
	result := make([]CuratedName, len(domainNames))
	for i, n := range domainNames {
		result[i] = CuratedName{
			Name:        n.Name,
			Pinyin:      n.Pinyin,
			Gender:      n.Gender,
			Source:      n.Source,
			Meaning:     n.Meaning,
			Wuxing:      n.Wuxing,
			YinyunScore: n.YinyunScore,
			Styles:      n.Styles,
			Tags:        n.Tags,
		}
	}
	return result, nil
}