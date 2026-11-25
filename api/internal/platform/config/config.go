// Package config 加载并校验服务运行配置，环境变量优先于默认值。
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	defaultAIBaseURL     = "https://api.openai.com/v1"
	defaultMaxConns      = 10
	defaultMaxIdleConns  = 5
	defaultLogLevel      = "info"
	defaultServerAddr    = ":8080"
	defaultTimeout       = 15 * time.Second
	defaultTokenExpiry   = 24 * time.Hour
	defaultStorageDriver = "local"
	defaultStorageRoot   = "./data"
	storageDriverS3      = "s3"
	storageDriverLocal   = "local"
	envPrefix            = "VERISTACK"
)

// AIConfig 描述模型适配器参数，未配置密钥时退化为本地确定性规则。
type AIConfig struct {
	APIKey  string // 模型接口密钥
	BaseURL string // 模型接口地址
	Model   string // 模型标识
}

// AuthConfig 描述会话令牌签名参数。
type AuthConfig struct {
	ExpiresIn time.Duration // 令牌有效期
	Secret    string        // 签名密钥
}

// SigningConfig 描述交付清单签名的密钥配置。
type SigningConfig struct {
	PrivateKeyPath string // Ed25519 私钥文件路径，为空时不签名
}

// Config 聚合所有配置分组。
type Config struct {
	AI       AIConfig       // 模型适配配置
	Auth     AuthConfig     // 会话令牌配置
	Database DatabaseConfig // 数据库连接配置
	LogLevel string         // 日志级别
	Server   ServerConfig   // HTTP 服务配置
	Signing  SigningConfig  // 交付清单签名配置
	Storage  StorageConfig  // 对象存储配置
	Webhook  WebhookConfig  // Webhook 接入配置
}

// WebhookConfig 描述第三方 Webhook 的校验密钥。
type WebhookConfig struct {
	GitHubSecret string // GitHub 事件签名密钥
	GitLabSecret string // GitLab 事件令牌密钥
	GiteeSecret  string // Gitee 事件令牌密钥
}

// DatabaseConfig 描述 PostgreSQL 连接参数。
type DatabaseConfig struct {
	DSN          string // PostgreSQL 连接串
	MaxConns     int    // 最大连接数
	MaxIdleConns int    // 最大空闲连接数
}

// ServerConfig 描述 HTTP 服务监听参数。
type ServerConfig struct {
	Addr         string        // 监听地址
	ReadTimeout  time.Duration // 请求读取超时
	WriteTimeout time.Duration // 响应写入超时
}

// StorageConfig 描述对象存储参数。
type StorageConfig struct {
	Driver    string // 存储驱动：local 或 s3
	LocalRoot string // 本地存储根目录
}

// Load 读取环境变量并构造配置，缺失必填项时返回错误。
func Load() (*Config, error) {
	v := newViper()
	cfg := &Config{
		AI: AIConfig{
			APIKey:  v.GetString("ai.apikey"),
			BaseURL: v.GetString("ai.baseurl"),
			Model:   v.GetString("ai.model"),
		},
		Auth: AuthConfig{
			ExpiresIn: v.GetDuration("auth.expiresin"),
			Secret:    v.GetString("auth.secret"),
		},
		Database: DatabaseConfig{
			DSN:          v.GetString("database.dsn"),
			MaxConns:     v.GetInt("database.maxconns"),
			MaxIdleConns: v.GetInt("database.maxidleconns"),
		},
		LogLevel: v.GetString("loglevel"),
		Server: ServerConfig{
			Addr:         v.GetString("server.addr"),
			ReadTimeout:  v.GetDuration("server.readtimeout"),
			WriteTimeout: v.GetDuration("server.writetimeout"),
		},
		Storage: StorageConfig{
			Driver:    v.GetString("storage.driver"),
			LocalRoot: v.GetString("storage.localroot"),
		},
		Webhook: WebhookConfig{
			GitHubSecret: v.GetString("webhook.githubsecret"),
			GitLabSecret: v.GetString("webhook.gitlabsecret"),
			GiteeSecret:  v.GetString("webhook.giteesecret"),
		},
		Signing: SigningConfig{
			PrivateKeyPath: v.GetString("signing.privatekeypath"),
		},
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate 校验必填配置，让错误在服务启动阶段以可读信息暴露。
func (c *Config) Validate() error {
	if c.Auth.Secret == "" {
		return fmt.Errorf("令牌签名密钥不能为空")
	}
	if c.Database.DSN == "" {
		return fmt.Errorf("数据库 DSN 不能为空")
	}
	if c.Storage.Driver != storageDriverLocal && c.Storage.Driver != storageDriverS3 {
		return fmt.Errorf("存储驱动必须为 %s 或 %s，当前为 %q", storageDriverLocal, storageDriverS3, c.Storage.Driver)
	}
	return nil
}

// newViper 创建配置读取器并注册默认值与环境变量前缀。
func newViper() *viper.Viper {
	v := viper.New()
	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	v.SetDefault("auth.expiresin", defaultTokenExpiry)
	v.SetDefault("ai.baseurl", defaultAIBaseURL)
	v.SetDefault("database.maxconns", defaultMaxConns)
	v.SetDefault("database.maxidleconns", defaultMaxIdleConns)
	v.SetDefault("loglevel", defaultLogLevel)
	v.SetDefault("server.addr", defaultServerAddr)
	v.SetDefault("server.readtimeout", defaultTimeout)
	v.SetDefault("server.writetimeout", defaultTimeout)
	v.SetDefault("storage.driver", storageDriverLocal)
	v.SetDefault("storage.localroot", defaultStorageRoot)
	return v
}
