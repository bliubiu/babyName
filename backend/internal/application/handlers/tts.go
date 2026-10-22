// Package handlers 内的离线粤语 TTS 处理器：代理到本地 Audio8 ONNX Runtime sidecar。
//
// 端点（挂在 /api/v1 下）：
//   - GET  /tts                  健康检查，返回 {"available": bool}
//   - POST /tts  {"jyutping":"wong4 zi2 man4"}  合成粤语读音 WAV
//
// 粤语发音正确性的关键：请求体喂的是 Jyutping 罗马字（含声调数字），而非汉字，
// 由前端 jyutping.ts 产出；Audio8 按文本发音，Jyutping 才能锁死粤语声调。
package handlers

import (
	"errors"
	"net/http"

	"name/internal/application/services"

	"github.com/gin-gonic/gin"
)

// TTSHandler 处理离线粤语 TTS 请求。
type TTSHandler struct {
	svc *services.TTSService
}

func NewTTSHandler(svc *services.TTSService) *TTSHandler {
	return &TTSHandler{svc: svc}
}

// Health GET /api/v1/tts —— 返回离线引擎是否就绪。
func (h *TTSHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"available": h.svc.Available()})
}

type ttsRequest struct {
	Jyutping string `json:"jyutping"`
}

// SynthesizeCantonese POST /api/v1/tts —— 合成粤语读音 WAV。
func (h *TTSHandler) SynthesizeCantonese(c *gin.Context) {
	var req ttsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体必须是 {\"jyutping\":\"...\"}", "code": "BAD_REQUEST"})
		return
	}
	if req.Jyutping == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "jyutping 不能为空", "code": "BAD_REQUEST"})
		return
	}

	audio, err := h.svc.SynthesizeCantonese(c.Request.Context(), req.Jyutping)
	if errors.Is(err, services.ErrTTSUnavailable) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "离线粤语 TTS 引擎未启用（未配置 AUDIO8_TTS_URL 或 sidecar 未启动）",
			"code":  "TTS_UNAVAILABLE",
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": "TTS_FAILED"})
		return
	}

	c.Header("Content-Type", "audio/wav")
	c.Header("Content-Disposition", "inline; filename=\"cantonese.wav\"")
	c.Data(http.StatusOK, "audio/wav", audio)
}
