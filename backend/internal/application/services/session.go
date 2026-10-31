package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"name/internal/application/validator"
)

// 会话标识用于隔离不同访客的收藏与历史记录。
//
// 设计边界：它是**随机 bearer 标识**，不是身份认证。持有同一标识即视为同一
// 访客，因此只通过 HttpOnly cookie 下发；API 客户端可用 X-Session-ID 头显式
// 携带。标识本身不含任何用户信息，也不参与权限判断——收藏/历史的读取与删除
// 始终以「记录归属的会话标识是否等于当前会话」为准。
const (
	// sessionIDBytes 新会话标识的随机字节数（128 位）。
	sessionIDBytes = 16
	// sessionIDMinLen/sessionIDMaxLen 接受的历史长度区间：旧版本 cookie 为
	// 16 位十六进制，新版本为 32 位；统一按十六进制校验，拒绝任意文本。
	sessionIDMinLen = 16
	sessionIDMaxLen = 64
)

// NewSessionID 生成 32 位十六进制（128 位随机）会话标识。
func NewSessionID() (string, error) {
	var b [sessionIDBytes]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("生成会话标识失败: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

// NormalizeSessionID 校验并归一化会话标识（去空白、转小写）。
//
// 非法输入返回校验错误，调用方不得把它降级为空串继续查询——空串会匹配到
// 迁移前的历史数据（session_id 默认 ”），等于重新暴露全局数据。
func NormalizeSessionID(sid string) (string, error) {
	sid = strings.ToLower(strings.TrimSpace(sid))
	if len(sid) < sessionIDMinLen || len(sid) > sessionIDMaxLen {
		return "", validator.NewValidationError("session", "会话标识无效")
	}
	for _, ch := range sid {
		if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') {
			continue
		}
		return "", validator.NewValidationError("session", "会话标识无效")
	}
	return sid, nil
}
