package config

import (
	"fmt"
	"os"
	"path/filepath"
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

var globalConfig *AppConfig

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

	// 自动绑定环境变量（前缀 NAME_）
	v.SetEnvPrefix("NAME")
	v.AutomaticEnv()

	// 加载配置文件
	if configPath == "" {
		// 尝试从多个位置查找配置文件
		v.SetConfigName("application")
		v.SetConfigType("yml")

		// 搜索路径：当前目录、可执行文件目录、项目根目录
		v.AddConfigPath(".")
		if exe, err := os.Executable(); err == nil {
			v.AddConfigPath(filepath.Dir(exe))
		}
		v.AddConfigPath("../")
	} else {
		v.SetConfigFile(configPath)
	}

	// 读取配置（如果文件不存在则使用默认值）
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("加载配置文件失败: %w", err)
		}
		// 配置文件不存在，使用默认值
	}

	var cfg AppConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 解析时间类型字段
	if cfg.Server.ReadHeaderTimeout == 0 {
		if d, err := time.ParseDuration(v.GetString("server.read_header_timeout")); err == nil {
			cfg.Server.ReadHeaderTimeout = d
		} else {
			cfg.Server.ReadHeaderTimeout = 10 * time.Second
		}
	}

	globalConfig = &cfg
	return &cfg, nil
}

// Get 获取全局配置
func Get() *AppConfig {
	if globalConfig == nil {
		// 如果未加载，尝试加载默认配置
		cfg, _ := Load("")
		return cfg
	}
	return globalConfig
}

// GetString 获取字符串配置值
func GetString(key string, defaultValue ...string) string {
	v := viper.GetViper()
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
	v := viper.GetViper()
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
	v := viper.GetViper()
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
	v := viper.GetViper()
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
	v := viper.GetViper()
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
