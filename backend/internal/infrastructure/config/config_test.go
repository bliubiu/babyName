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

// TestEnvOverrideWithReplacer 环境变量覆盖嵌套键（docs/29 B13 回归）。
// AutomaticEnv 对 server.port 默认查 NAME_SERVER.PORT（保留点号），任何
// shell 都设不出来，覆盖静默无效；必须配 SetEnvKeyReplacer 把「.」映射
// 为「_」，NAME_SERVER_PORT 才能生效。修复前本用例失败（读到文件值 9999）。
func TestEnvOverrideWithReplacer(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "application.yml")
	content := `
server:
  host: 0.0.0.0
  port: 9999
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("写配置失败: %v", err)
	}

	t.Setenv("NAME_SERVER_PORT", "7777")

	if _, err := Load(cfgPath); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if got := GetInt("server.port", 0); got != 7777 {
		t.Errorf("环境变量 NAME_SERVER_PORT 应覆盖文件值：got %d，want 7777（文件值 9999）", got)
	}
	// 未被覆盖的键仍读文件值
	if got := GetString("server.host", ""); got != "0.0.0.0" {
		t.Errorf("未覆盖键应读文件值 0.0.0.0，got %q", got)
	}
}

// TestLoad_SearchesConfigSubDir 配置文件放在 ./config/ 下时必须被找到
//
// docs/29 B12：compose 把宿主机配置目录挂到容器的 /app/config，而进程 cwd 是
// /app。原搜索路径只有 "." / exe 目录 / "../"，全部落不到 /app/config，
// 于是配置文件被静默跳过、整套默认值生效（监听 localhost、日志与数据库写到
// 非挂载目录）。这类故障无任何报错，极难定位。
func TestLoad_SearchesConfigSubDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "config"), 0755); err != nil {
		t.Fatalf("建目录失败: %v", err)
	}
	cfgPath := filepath.Join(dir, "config", "application.yml")
	content := "server:\n  host: 0.0.0.0\n  port: 9123\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("写配置失败: %v", err)
	}

	// 在临时目录内执行，使 "." 指向不含配置的父目录
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("取工作目录失败: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("切换工作目录失败: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	// 清掉可能干扰的环境变量
	t.Setenv("NAME_SERVER_HOST", "")
	t.Setenv("NAME_SERVER_PORT", "")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Server.Port != 9123 {
		t.Errorf("应从 ./config/application.yml 读到 port=9123，实际 %d（说明搜索路径漏了 config 子目录）", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("应从 ./config/application.yml 读到 host=0.0.0.0，实际 %q", cfg.Server.Host)
	}
}

// TestLoad_ConfigDirEnv 自定义 CONFIG_DIR 必须生效
func TestLoad_ConfigDirEnv(t *testing.T) {
	base := t.TempDir()
	cfgDir := filepath.Join(base, "custom-conf")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("建目录失败: %v", err)
	}
	content := "database:\n  path: /app/storage/namer.db\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "application.yml"), []byte(content), 0644); err != nil {
		t.Fatalf("写配置失败: %v", err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("取工作目录失败: %v", err)
	}
	if err := os.Chdir(base); err != nil {
		t.Fatalf("切换工作目录失败: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	t.Setenv("CONFIG_DIR", cfgDir)
	t.Setenv("NAME_DATABASE_PATH", "")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Database.Path != "/app/storage/namer.db" {
		t.Errorf("CONFIG_DIR 应生效，database.path=%q", cfg.Database.Path)
	}
}

// TestEnvOverrideReachesStructFields 环境变量必须穿透到结构体字段
// （docs/30 §六 C10 回归）：viper 的 Unmarshal 不解析 AutomaticEnv——
// compose 传入的 NAME_MODE/NAME_SERVER_HOST 等此前对 cfg.Mode 等字段
// 静默失效（mode 保持 "all" 使容器内静态目录 Fatal）。修复后逐字段
// 经 viper Get 重解析。
func TestEnvOverrideReachesStructFields(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "application.yml")
	content := `
mode: all
server:
  host: localhost
  port: 8080
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("写配置失败: %v", err)
	}

	t.Setenv("NAME_MODE", "api")
	t.Setenv("NAME_SERVER_HOST", "0.0.0.0")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Mode != "api" {
		t.Errorf("NAME_MODE 应覆盖 yml 的 all：got %q，want api", cfg.Mode)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("NAME_SERVER_HOST 应覆盖 yml 的 localhost：got %q，want 0.0.0.0", cfg.Server.Host)
	}
	// 未被 env 覆盖的键仍读文件值
	if cfg.Server.Port != 8080 {
		t.Errorf("未覆盖键应读文件值 8080，got %d", cfg.Server.Port)
	}
}
