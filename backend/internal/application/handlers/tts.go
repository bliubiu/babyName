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
	"context"
	"errors"
	"net/http"
	"time"

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

// ttsRequestTimeout 离线粤语 TTS 的合成超时。
// 全局 RequestTimeout 中间件（cmd/server/main.go）按同步起名端点标定为 30s，
// 而 Audio8 CPU 推理单条实测 44-49s（max_new_tokens=256），继承请求 deadline
// 会被误杀（客户端收到 503 TTS_UNAVAILABLE，sidecar 却仍在后台跑完合成）。
// 故此处用独立上下文调用 sidecar；副作用是客户端断连不会中断合成（本地单用户
// sidecar 可接受，且上限 300s 与 SidecarTTSSynthesizer 的 HTTP 客户端超时一致）。
var ttsRequestTimeout = 300 * time.Second

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

	ctx, cancel := context.WithTimeout(context.Background(), ttsRequestTimeout)
	defer cancel()

	audio, err := h.svc.SynthesizeCantonese(ctx, req.Jyutping)
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
