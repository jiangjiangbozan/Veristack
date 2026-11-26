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
	minSecretLength      = 32
	defaultSMTPPort      = 587
	defaultSMTPTimeout   = 10 * time.Second
)

// insecureSecrets 是示例与历史默认值，出现即拒绝启动。
var insecureSecrets = []string{
	"change-me-in-production",
	"changeme",
	"secret",
	"veristack",
}

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
	SMTP     SMTPConfig     // 邮件发送通道
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
		SMTP: SMTPConfig{
			Enabled:  v.GetBool("smtp.enabled"),
			From:     v.GetString("smtp.from"),
			FromName: v.GetString("smtp.fromname"),
			Host:     v.GetString("smtp.host"),
			Password: v.GetString("smtp.password"),
			Port:     v.GetInt("smtp.port"),
			Timeout:  v.GetDuration("smtp.timeout"),
			TLS:      v.GetBool("smtp.tls"),
			Username: v.GetString("smtp.username"),
		},
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
//
// 配置采用 Fail Closed 原则：已知默认值、过弱密钥与尚未实现的驱动都必须拒绝启动，
// 避免服务在「看起来正常」的状态下用错误的安全配置对外提供服务。
func (c *Config) Validate() error {
	if c.Auth.Secret == "" {
		return fmt.Errorf("令牌签名密钥不能为空")
	}
	if err := ValidateSecretStrength(c.Auth.Secret); err != nil {
		return err
	}
	if c.Database.DSN == "" {
		return fmt.Errorf("数据库 DSN 不能为空")
	}
	if c.Storage.Driver != storageDriverLocal {
		return fmt.Errorf("存储驱动 %q 尚未实现，当前仅支持 %s", c.Storage.Driver, storageDriverLocal)
	}
	if c.Storage.LocalRoot == "" {
	}
	if err := c.SMTP.Validate(); err != nil {
		return err
	}
	if c.Storage.LocalRoot == "" {
		return fmt.Errorf("本地存储根目录不能为空")
	}
	return nil
}

// ValidateSecretStrength 校验密钥强度，拒绝已知默认值与过短密钥。
func ValidateSecretStrength(secret string) error {
	normalized := strings.ToLower(strings.TrimSpace(secret))
	for _, weak := range insecureSecrets {
		if normalized == weak {
			return fmt.Errorf("令牌签名密钥仍是示例默认值，必须改为随机值")
		}
	}
	if len(secret) < minSecretLength {
		return fmt.Errorf("令牌签名密钥长度不足 %d 位，当前 %d 位", minSecretLength, len(secret))
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
	v.SetDefault("smtp.port", defaultSMTPPort)
	v.SetDefault("smtp.timeout", defaultSMTPTimeout)
	v.SetDefault("storage.driver", storageDriverLocal)
	v.SetDefault("storage.localroot", defaultStorageRoot)
	return v
}
