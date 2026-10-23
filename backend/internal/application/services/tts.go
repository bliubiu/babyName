// Package services 内的 TTS 合成服务：把「汉字原名」交给本地离线粤语引擎
// （Audio8 0.1B INT8 ONNX Runtime sidecar）合成为 WAV。
//
// 设计要点：
//   - 离线优先：保证断网也能朗读；但 Audio8 无 Jyutping→音素转换层，
//     必须由调用方传入汉字原名（喂粤拼罗马字会被当英文字母+数字乱读成噪音）。
//   - 降级：sidecar 未配置/未启动时返回 ErrTTSUnavailable，处理器据此回 503，
//     前端回退到 Web Speech（零成本兜底，参见 frontend/src/lib/jyutping.ts）。
//   - sidecar 即 Audio8 官方 onnx_runtime_0_1b_int8 起的服务
//     （默认 127.0.0.1:8024/api/tts，强制 CPUExecutionProvider，约 0.6GB 内存）。
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

// ErrTTSUnavailable 表示离线粤语 TTS 引擎未启用（未配置 AUDIO8_TTS_URL 或 sidecar 不可达）。
var ErrTTSUnavailable = errors.New("offline Cantonese TTS engine unavailable")

// TTSSynthesizer 把文本（汉字原名）合成为音频字节。
type TTSSynthesizer interface {
	// SynthesizeCantonese 返回 WAV 音频字节；不可用或失败时返回错误。
	SynthesizeCantonese(ctx context.Context, jyutping string) ([]byte, error)
	// Available 指示该合成器是否就绪（用于 /tts 健康检查）。
	Available() bool
}

// SidecarTTSSynthesizer 通过 HTTP 调用本地 Audio8 ONNX Runtime 服务。
type SidecarTTSSynthesizer struct {
	BaseURL      string
	VoiceName    string
	MaxNewTokens int
	HTTPClient   *http.Client
}

func (s *SidecarTTSSynthesizer) Available() bool { return true }

func (s *SidecarTTSSynthesizer) SynthesizeCantonese(ctx context.Context, jyutping string) ([]byte, error) {
	body, err := json.Marshal(audio8TTSRequest{Text: jyutping, VoiceName: s.VoiceName, MaxNewTokens: s.MaxNewTokens})
	if err != nil {
		return nil, fmt.Errorf("marshal tts request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.BaseURL+"/api/tts", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build tts request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: sidecar unreachable: %v", ErrTTSUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusServiceUnavailable {
		return nil, ErrTTSUnavailable
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tts sidecar returned %d", resp.StatusCode)
	}
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, fmt.Errorf("read tts audio: %w", err)
	}
	return buf.Bytes(), nil
}

// audio8TTSRequest 对应 Audio8 /api/tts 的请求体。
type audio8TTSRequest struct {
	Text      string `json:"text"`
	VoiceName string `json:"voice_name"`
	// MaxNewTokens 限制自回归生成步数：CPU 推理下单步约 0.2-0.3s，
	// 不设上限时 sidecar 默认 1024 步，短文本也可能超过 120s（实测）。
	// 粤拼短语（名字读音）256 步足够（实测 ~49s/条）。
	MaxNewTokens int `json:"max_new_tokens,omitempty"`
}

// DisabledTTSSynthesizer 离线引擎未配置时的占位实现。
type DisabledTTSSynthesizer struct{}

func (DisabledTTSSynthesizer) Available() bool { return false }
func (DisabledTTSSynthesizer) SynthesizeCantonese(context.Context, string) ([]byte, error) {
	return nil, ErrTTSUnavailable
}

// TTSService 聚合合成器，供处理器调用。
type TTSService struct {
	synth TTSSynthesizer
}

func (s *TTSService) Available() bool { return s.synth.Available() }

func (s *TTSService) SynthesizeCantonese(ctx context.Context, jyutping string) ([]byte, error) {
	return s.synth.SynthesizeCantonese(ctx, jyutping)
}

// NewTTSService 按环境变量装配合成器。
//
//   - AUDIO8_TTS_URL：Audio8 ONNX Runtime 服务基址（sidecar 约定监听 http://127.0.0.1:8024）。
//     未设置则该服务视为未启用（降级到 Web Speech）——显式 opt-in，避免 sidecar 缺席时
//     健康检查误报可用（SidecarTTSSynthesizer.Available 不探测 sidecar 存活）。
//   - AUDIO8_TTS_VOICE：音色名（默认 default）。
//   - AUDIO8_TTS_MAX_TOKENS：自回归生成步数上限（默认 256）。CPU 推理单步约 0.2-0.3s，
//     汉字名字读音 256 步实测约 49s；不设上限（sidecar 默认 1024）时短文本也可能超过 120s。
//   - HTTP 客户端超时固定 300s：覆盖 CPU 上 256 步合成的上限，留足余量。
func NewTTSService() *TTSService {
	base := os.Getenv("AUDIO8_TTS_URL")
	if base == "" {
		return &TTSService{synth: DisabledTTSSynthesizer{}}
	}
	voice := os.Getenv("AUDIO8_TTS_VOICE")
	if voice == "" {
		voice = "default"
	}
	maxTokens := 256
	if v := os.Getenv("AUDIO8_TTS_MAX_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxTokens = n
		}
	}
	return &TTSService{synth: &SidecarTTSSynthesizer{
		BaseURL:       base,
		VoiceName:     voice,
		MaxNewTokens:  maxTokens,
		HTTPClient:    &http.Client{Timeout: 300 * time.Second},
	}}
}
