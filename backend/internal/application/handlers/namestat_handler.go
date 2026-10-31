package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"name/internal/application/response"
	"name/internal/domain/namestat"
	"name/internal/infrastructure/logger"
)

type NameStatHandler struct{}

func NewNameStatHandler() *NameStatHandler {
	return &NameStatHandler{}
}

// GetNameStats 获取名字统计信息
// @Summary 获取名字统计信息
// @Description 根据名字获取相关统计信息
// @Tags 名字统计
// @Accept json
// @Produce json
// @Param name path string true "名字"
// @Success 200 {object} namestat.NameStat "成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /namestat/{name} [get]
func (h *NameStatHandler) GetNameStats(c *gin.Context) {
	name := c.Param("name")
	logger.Info("GetNameStats called", zap.String("name", name))
	stats := namestat.GetNameStats(name)
	response.SuccessJSON(c, stats)
}

// GetTopNames 获取热门名字列表
// @Summary 获取热门名字列表
// @Description 获取热门名字排行榜
// @Tags 名字统计
// @Accept json
// @Produce json
// @Param limit query int false "限制数量，默认20，最大100"
// @Param kind query string false "榜单种类：all(默认，合并)/char(仅单字名)/bigram(仅双字名)"
// @Success 200 {array} namestat.NameStat "成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /namestat [get]
func (h *NameStatHandler) GetTopNames(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	kind := c.Query("kind")
	if kind != "char" && kind != "bigram" {
		kind = "all"
	}

	logger.Info("GetTopNames called", zap.Int("limit", limit), zap.String("kind", kind))
	stats := namestat.GetTopNames(limit, kind)
	response.SuccessJSON(c, stats)
}
