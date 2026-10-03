package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// AppConfig 应用配置结构
type AppConfig struct {
	Server    ServerConfig    `mapstructure:"server"`
	Mode      string          `mapstructure:"mode"`
	Static    string          `mapstructure:"static"`
	Log       LogConfig       `mapstructure:"log"`
	Database  DatabaseConfig  `mapstructure:"database"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	Redis     RedisConfig     `mapstructure:"redis"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host              string        `mapstructure:"host"`
	Port              int           `mapstructure:"port"`
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	OutputPath string `mapstructure:"output_path"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver string `mapstructure:"driver"`
	Path   string `mapstructure:"path"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Capacity float64 `mapstructure:"capacity"`
	Rate     float64 `mapstructure:"rate"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

var (
	globalConfig *AppConfig

	// globalViper 保存 Load() 实际使用的配置实例。
	//
	// ★ 历史缺陷（docs/28 W2）：Load() 用 viper.New() 创建**局部**实例，
	// 而 GetString/GetInt/GetFloat64/GetBool/GetDuration 五个 getter 读的是
	// viper.GetViper() —— 那是 viper 的**包级全局单例**，与局部实例毫无关系。
	// 后果：这五个 getter 恒定返回 viper 默认值（空串/0/false），配置文件里
	// 写了什么、环境变量设了什么，一概读不到；且因 viper 全局单例从未被写入，
	// 连默认值也不会有（v.SetDefault 只作用于局部实例）。
	//
	// 现统一为：Load() 把实例存进此处，getter 读它，内部走同一份配置来源。
	// 同时显式把 viper 全局单例也指向同一实例（SetDefault 反向同步无法做到，
	// 故 getter 一律走 globalViper，不再依赖全局单例）。
	globalViper *viper.Viper

	// configMu 保护 globalConfig / globalViper 的读写：
	// Load 可能被 main 与测试并发调用（config.Get() 的懒加载分支亦会触发）。
	configMu sync.RWMutex
)

// Load 加载配置文件
// configPath: 配置文件路径，如果为空则使用默认路径
func Load(configPath string) (*AppConfig, error) {
	v := viper.New()

	// 设置默认值
	v.SetDefault("server.host", "localhost")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_header_timeout", "10s")
	v.SetDefault("mode", "all")
	v.SetDefault("static", "")
	v.SetDefault("log.level", "debug")
	v.SetDefault("log.format", "console")
	v.SetDefault("log.output_path", "namer.log")
	v.SetDefault("database.driver", "sqlite")
	v.SetDefault("database.path", "namer.db")
	v.SetDefault("rate_limit.capacity", 100)
	v.SetDefault("rate_limit.rate", 10)
	v.SetDefault("redis.enabled", false)
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	// 自动绑定环境变量（前缀 NAME_）。
	// 必须配 SetEnvKeyReplacer 把嵌套键的「.」映射为「_」：AutomaticEnv 对
	// server.port 查找的环境变量名是 NAME_SERVER.PORT（保留点号），任何
	// shell 都设不出来，覆盖静默无效（docs/29 B13）。替换后 NAME_SERVER_PORT
	// 即可覆盖 server.port。
	v.SetEnvPrefix("NAME")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 加载配置文件
	if configPath == "" {
		// 尝试从多个位置查找配置文件
		v.SetConfigName("application")
		v.SetConfigType("yml")

		// 搜索路径：当前目录、可执行文件目录、项目根目录、config 子目录、
		// 以及 CONFIG_DIR 指定目录。
		//
		// 最后两项是容器部署的必需项（docs/29 B12）：compose 把宿主机配置目录
		// 挂到 /app/config，而进程 cwd 是 /app —— 原搜索路径（"." / exe 目录 /
		// "../"）全部落不到 /app/config，于是配置文件被静默跳过、整套默认值生效：
		// 监听 localhost（容器外不可达）、日志写到容器内相对路径（重启即丢）、
		// 数据库写到非挂载目录（重建容器数据全丢）。这类故障没有任何报错，
		// 只会表现为「服务起来了但什么都不对」。
		v.AddConfigPath(".")
		if exe, err := os.Executable(); err == nil {
			v.AddConfigPath(filepath.Dir(exe))
		}
		v.AddConfigPath("../")
		v.AddConfigPath("config")
		// CONFIG_DIR 优先，便于自定义镜像布局；为空则忽略
		if dir := os.Getenv("CONFIG_DIR"); dir != "" {
			v.AddConfigPath(dir)
		}
	} else {
		v.SetConfigFile(configPath)
	}

	// 读取配置（如果文件不存在则使用默认值）
	//
	// ★ 注意 viper 两种模式的错误类型不同（历史缺陷，本次一并修复）：
	//   - SetConfigName + AddConfigPath 模式（未指定路径）：文件找不到返回
	//     viper.ConfigFileNotFoundError —— 这是「正常回落默认值」的信号；
	//   - SetConfigFile 模式（显式指定路径）：文件找不到返回 *fs.PathError，
	//     **不是** ConfigFileNotFoundError。
	// 原实现只判断 ConfigFileNotFoundError，导致「显式指定了一个不存在的配置
	// 路径」会直接 fatal 报错退出，而不是按注释承诺的「使用默认值」。
	// 现两种都视为可回落的缺文件情形。
	if err := v.ReadInConfig(); err != nil {
		if !isConfigNotFound(err, configPath) {
			return nil, fmt.Errorf("加载配置文件失败: %w", err)
		}
		// 配置文件不存在，使用默认值
	}

	var cfg AppConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// ★ viper 的 Unmarshal 不解析 AutomaticEnv 环境变量：键已存在于 settings
	// map（有默认值或配置文件提供）时按 map 值解码，NAME_* 环境变量覆盖对
	// 结构体字段**静默失效**（docs/30 §六 C10——compose 传入的
	// NAME_MODE=api / NAME_SERVER_HOST=0.0.0.0 等全部落空，mode 保持 "all"
	// 使容器内静态目录 Fatal）。逐字段经 viper Get 重新解析：
	// Get 走 AutomaticEnv + SetEnvKeyReplacer，环境变量优先，无 env 时
	// 回落配置文件/默认值，语义不变。
	cfg.Mode = v.GetString("mode")
	cfg.Static = v.GetString("static")
	cfg.Server.Host = v.GetString("server.host")
	cfg.Server.Port = v.GetInt("server.port")
	cfg.Log.Level = v.GetString("log.level")
	cfg.Log.Format = v.GetString("log.format")
	cfg.Log.OutputPath = v.GetString("log.output_path")
	cfg.Database.Driver = v.GetString("database.driver")
	cfg.Database.Path = v.GetString("database.path")
	cfg.RateLimit.Capacity = v.GetFloat64("rate_limit.capacity")
	cfg.RateLimit.Rate = v.GetFloat64("rate_limit.rate")
	cfg.Redis.Enabled = v.GetBool("redis.enabled")
	cfg.Redis.Addr = v.GetString("redis.addr")
	cfg.Redis.Password = v.GetString("redis.password")
	cfg.Redis.DB = v.GetInt("redis.db")

	// 解析时间类型字段
	if cfg.Server.ReadHeaderTimeout == 0 {
		if d, err := time.ParseDuration(v.GetString("server.read_header_timeout")); err == nil {
			cfg.Server.ReadHeaderTimeout = d
		} else {
			cfg.Server.ReadHeaderTimeout = 10 * time.Second
		}
	}

	configMu.Lock()
	globalConfig = &cfg
	globalViper = v
	configMu.Unlock()
	return &cfg, nil
}

// getViper 返回当前生效的配置实例；未加载时返回 nil。
// 所有 getter 都必须经由此处取实例，禁止直接使用 viper.GetViper() 全局单例。
func getViper() *viper.Viper {
	configMu.RLock()
	defer configMu.RUnlock()
	return globalViper
}

// isConfigNotFound 判断读取配置的错误是否属于「配置文件不存在」（应回落默认值），
// 而非真正的解析/权限错误（应向上报错）。
//
// viper 在两种加载模式下给出不同类型的错误：
//   - SetConfigName/AddConfigPath 模式 → viper.ConfigFileNotFoundError
//   - SetConfigFile 模式               → *fs.PathError（errors.Is(err, fs.ErrNotExist)）
func isConfigNotFound(err error, configPath string) bool {
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		return true
	}
	// 显式指定路径时，仅当该路径确实不存在才回落；其余错误一律上报。
	if configPath != "" && errors.Is(err, fs.ErrNotExist) {
		return true
	}
	return false
}

// Get 获取全局配置
func Get() *AppConfig {
	configMu.RLock()
	cfg := globalConfig
	configMu.RUnlock()
	if cfg != nil {
		return cfg
	}
	// 未加载时惰性加载默认配置（Load 内部会写回 globalConfig/globalViper）
	loaded, err := Load("")
	if err != nil || loaded == nil {
		return &AppConfig{}
	}
	return loaded
}

// GetString 获取字符串配置值
func GetString(key string, defaultValue ...string) string {
	v := getViper()
	if v == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return ""
	}
	val := v.GetString(key)
	if val == "" && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return val
}

// GetInt 获取整数配置值
func GetInt(key string, defaultValue ...int) int {
	v := getViper()
	if v == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	val := v.GetInt(key)
	if val == 0 && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return val
}

// GetFloat64 获取浮点数配置值
func GetFloat64(key string, defaultValue ...float64) float64 {
	v := getViper()
	if v == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	val := v.GetFloat64(key)
	if val == 0 && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return val
}

// GetBool 获取布尔配置值
func GetBool(key string, defaultValue ...bool) bool {
	v := getViper()
	if v == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return false
	}
	return v.GetBool(key)
}

// GetDuration 获取时间配置值
func GetDuration(key string, defaultValue ...time.Duration) time.Duration {
	v := getViper()
	if v == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	val := v.GetDuration(key)
	if val == 0 && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return val
}
