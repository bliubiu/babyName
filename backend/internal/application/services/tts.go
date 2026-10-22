// Package services 内的 TTS 合成服务：把粤语 Jyutping 串交给本地离线粤语引擎
// （Audio8 0.1B INT8 ONNX Runtime sidecar）合成为 WAV。
//
// 设计要点：
//   - 离线优先：保证断网也能朗读，且粤语发音正确（喂 Jyutping 罗马字而非汉字）。
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
	"time"
)

// ErrTTSUnavailable 表示离线粤语 TTS 引擎未启用（未配置 AUDIO8_TTS_URL 或 sidecar 不可达）。
var ErrTTSUnavailable = errors.New("offline Cantonese TTS engine unavailable")

// TTSSynthesizer 把文本（粤语 Jyutping 罗马字）合成为音频字节。
type TTSSynthesizer interface {
	// SynthesizeCantonese 返回 WAV 音频字节；不可用或失败时返回错误。
	SynthesizeCantonese(ctx context.Context, jyutping string) ([]byte, error)
	// Available 指示该合成器是否就绪（用于 /tts 健康检查）。
	Available() bool
}

// SidecarTTSSynthesizer 通过 HTTP 调用本地 Audio8 ONNX Runtime 服务。
type SidecarTTSSynthesizer struct {
	BaseURL    string
	VoiceName  string
	HTTPClient *http.Client
}

func (s *SidecarTTSSynthesizer) Available() bool { return true }

func (s *SidecarTTSSynthesizer) SynthesizeCantonese(ctx context.Context, jyutping string) ([]byte, error) {
	body, err := json.Marshal(audio8TTSRequest{Text: jyutping, VoiceName: s.VoiceName})
	if err != nil {
		return nil, fmt.Errorf("marshal tts request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.BaseURL+"/api/tts", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build tts request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
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
//   - AUDIO8_TTS_URL：Audio8 ONNX Runtime 服务基址（默认 http://127.0.0.1:8024）。
//     未设置则该服务视为未启用（降级到 Web Speech）。
//   - AUDIO8_TTS_VOICE：音色名（默认 default）。
func NewTTSService() *TTSService {
	base := os.Getenv("AUDIO8_TTS_URL")
	if base == "" {
		return &TTSService{synth: DisabledTTSSynthesizer{}}
	}
	voice := os.Getenv("AUDIO8_TTS_VOICE")
	if voice == "" {
		voice = "default"
	}
	return &TTSService{synth: &SidecarTTSSynthesizer{
		BaseURL:    base,
		VoiceName:  voice,
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
	}}
}
