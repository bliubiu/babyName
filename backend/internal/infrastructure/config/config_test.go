package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestGettersReadLoadedConfig 验证 5 个 getter 读的是 Load() 实际加载的配置实例。
//
// ★ 历史缺陷（docs/28 W2）：Load() 用 viper.New() 建**局部**实例，而
// GetString/GetInt/GetFloat64/GetBool/GetDuration 读的是 viper.GetViper()
// **包级全局单例**——两者毫无关联。后果是这 5 个 getter 恒返回默认值：
// 配置文件里写 `server.port: 9999`、`rate_limit.capacity: 250`，
// GetInt("server.port") 依然返回 0；连 v.SetDefault 设置的默认值也读不到
// （SetDefault 只作用于局部实例，全局单例从未被写入）。
//
// 本用例写一份真实的 yml 配置再 Load，断言 getter 能读回**文件里的值**。
// 修复前：全部断言失败（返回 0/""/false）。
func TestGettersReadLoadedConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "application.yml")
	content := `
server:
  host: 0.0.0.0
  port: 9999
mode: backend
log:
  level: warn
rate_limit:
  capacity: 250
  rate: 12.5
redis:
  enabled: true
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("写测试配置失败: %v", err)
	}

	if _, err := Load(cfgPath); err != nil {
		t.Fatalf("Load 失败: %v", err)
	}

	if got := GetString("server.host"); got != "0.0.0.0" {
		t.Errorf("GetString(server.host) = %q，期望 0.0.0.0（修复前为空串）", got)
	}
	if got := GetInt("server.port"); got != 9999 {
		t.Errorf("GetInt(server.port) = %d，期望 9999（修复前为 0）", got)
	}
	if got := GetString("mode"); got != "backend" {
		t.Errorf("GetString(mode) = %q，期望 backend", got)
	}
	if got := GetString("log.level"); got != "warn" {
		t.Errorf("GetString(log.level) = %q，期望 warn", got)
	}
	// 注意各 getter 的兜底语义：值为零值时返回 defaultValue。
	// capacity=250 非零，可直接断言读回文件值。
	if got := GetFloat64("rate_limit.capacity"); got != 250 {
		t.Errorf("GetFloat64(rate_limit.capacity) = %v，期望 250（修复前为 0）", got)
	}
	if got := GetFloat64("rate_limit.rate"); got != 12.5 {
		t.Errorf("GetFloat64(rate_limit.rate) = %v，期望 12.5（修复前为 0）", got)
	}
	if got := GetBool("redis.enabled"); !got {
		t.Errorf("GetBool(redis.enabled) = false，期望 true（修复前恒为 false）")
	}
}

// TestGettersReadDefaultsWhenNoFile 验证无配置文件时 getter 仍能读到
// Load() 内置的 SetDefault 值（这些默认值同样只存在于局部实例上）。
//
// 修复前：配置文件的 SetDefault 全部写在局部实例上，全局单例一片空白，
// 因此 GetInt("server.port") 返回 0 而非 8080。
func TestGettersReadDefaultsWhenNoFile(t *testing.T) {
	// 指向一个不存在的路径，触发「配置文件不存在 → 使用默认值」分支
	missing := filepath.Join(t.TempDir(), "no-such-config.yml")
	if _, err := Load(missing); err != nil {
		t.Fatalf("Load 不存在的文件不应报错（应回落默认值），实际: %v", err)
	}

	if got := GetInt("server.port"); got != 8080 {
		t.Errorf("GetInt(server.port) = %d，期望默认值 8080", got)
	}
	if got := GetString("server.host"); got != "localhost" {
		t.Errorf("GetString(server.host) = %q，期望默认值 localhost", got)
	}
	if got := GetString("log.level"); got != "debug" {
		t.Errorf("GetString(log.level) = %q，期望默认值 debug", got)
	}
	if got := GetFloat64("rate_limit.capacity"); got != 100 {
		t.Errorf("GetFloat64(rate_limit.capacity) = %v，期望默认值 100", got)
	}
	if got := GetBool("redis.enabled"); got {
		t.Errorf("GetBool(redis.enabled) = true，期望默认值 false")
	}
	if got := GetString("database.driver"); got != "sqlite" {
		t.Errorf("GetString(database.driver) = %q，期望默认值 sqlite", got)
	}
}

// TestGetDurationReadsConfig 验证 GetDuration 的取值与兜底。
func TestGetDurationReadsConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "application.yml")
	if err := os.WriteFile(cfgPath, []byte("server:\n  read_header_timeout: 25s\n"), 0o644); err != nil {
		t.Fatalf("写测试配置失败: %v", err)
	}
	if _, err := Load(cfgPath); err != nil {
		t.Fatalf("Load 失败: %v", err)
	}

	if got := GetDuration("server.read_header_timeout"); got != 25*time.Second {
		t.Errorf("GetDuration = %v，期望 25s（修复前为 0）", got)
	}

	// 键不存在时走 defaultValue 兜底
	if got := GetDuration("no.such.key", 7*time.Second); got != 7*time.Second {
		t.Errorf("GetDuration 兜底 = %v，期望 7s", got)
	}
}

// TestGetStringFallbackToDefaultValue 验证「配置值为空串时回落 defaultValue」的既有语义未被破坏。
func TestGetStringFallbackToDefaultValue(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.yml")
	if _, err := Load(missing); err != nil {
		t.Fatalf("Load 失败: %v", err)
	}
	// static 默认值为空串 → 应回落到调用方给的兜底值
	if got := GetString("static", "/fallback"); got != "/fallback" {
		t.Errorf("GetString(static, /fallback) = %q，期望 /fallback", got)
	}
	// 未设置且无兜底 → 空串
	if got := GetString("totally.unknown.key"); got != "" {
		t.Errorf("GetString(未知键) = %q，期望空串", got)
	}
}

// TestGetReturnsLoadedConfig 验证 Get() 返回的是 Load() 加载的那份配置对象。
func TestGetReturnsLoadedConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "application.yml")
	if err := os.WriteFile(cfgPath, []byte("server:\n  port: 4321\nmode: backend\n"), 0o644); err != nil {
		t.Fatalf("写测试配置失败: %v", err)
	}
	loaded, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load 失败: %v", err)
	}
	got := Get()
	if got == nil {
		t.Fatal("Get() 返回 nil")
	}
	if got.Server.Port != 4321 {
		t.Errorf("Get().Server.Port = %d，期望 4321", got.Server.Port)
	}
	if got != loaded {
		t.Error("Get() 应返回 Load() 的同一实例")
	}
}
