package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"name/internal/infrastructure/database"
)

// 分享快照服务端约束（2026-11-10 结果页动作区接线）
//
// 项目是匿名工具、无账号体系：分享链接任何人拿到 id 都能读，因此服务端
// 不能直接透存客户端传来的任意 JSON。应用层做两件事：
//  1. 字段白名单：只保留展示用的结论字段，剔除出生地/长辈姓名等输入信息；
//  2. 大小上限：避免把 shares 表当文件存储，也配合 main.go 的 1MB 请求体上限。
const maxSharePayloadBytes = 64 << 10

var (
	ErrShareNotFound = errors.New("分享不存在或已失效")
	ErrInvalidShare  = errors.New("分享内容不合法")
)

var shareIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

// shareAllowedFields 分享快照允许的顶层字段。前端构造快照时必须用这些键，
// 其余字段一律丢弃（不是报错）：老版本前端多带一个字段不应让分享直接失败。
var shareAllowedFields = map[string]bool{
	"title":         true,
	"surname":       true,
	"gender":        true,
	"rishou":        true,
	"xiyongshen":    true,
	"zodiac":        true,
	"hexagram_name": true,
	"five_elements": true,
	"names":         true,
	"total":         true,
	"generated_at":  true,
}

type ShareService struct {
	store database.ShareStore
}

func NewShareService(store database.ShareStore) *ShareService {
	return &ShareService{store: store}
}

func newShareID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成分享标识失败: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// Create 写入一份裁剪后的分享快照，返回可公开读取的随机 id。
func (s *ShareService) Create(ctx context.Context, payload json.RawMessage) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if len(payload) == 0 || len(payload) > maxSharePayloadBytes {
		return "", fmt.Errorf("%w: 内容为空或超过 %d 字节", ErrInvalidShare, maxSharePayloadBytes)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(payload, &obj); err != nil || obj == nil {
		return "", fmt.Errorf("%w: 必须是 JSON 对象", ErrInvalidShare)
	}
	for key := range obj {
		if !shareAllowedFields[key] {
			delete(obj, key)
		}
	}
	if len(obj) == 0 {
		return "", fmt.Errorf("%w: 没有可分享的内容", ErrInvalidShare)
	}
	cleaned, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("%w: 序列化失败", ErrInvalidShare)
	}
	id, err := newShareID()
	if err != nil {
		return "", err
	}
	if err := s.store.SaveShare(&database.ShareRecord{
		ID:        id,
		Payload:   string(cleaned),
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return "", err
	}
	return id, nil
}

// Get 读取分享快照；id 非法或不存在统一返回 ErrShareNotFound（不泄露存在性）。
func (s *ShareService) Get(ctx context.Context, id string) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !shareIDPattern.MatchString(id) {
		return nil, ErrShareNotFound
	}
	record, err := s.store.GetShare(id)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, ErrShareNotFound
	}
	return json.RawMessage(record.Payload), nil
}
