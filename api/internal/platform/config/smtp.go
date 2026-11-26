package config

import (
	"fmt"
	"time"
)

// SMTPConfig 描述邮件发送通道参数。
type SMTPConfig struct {
	Host     string        // SMTP 服务器地址
	Port     int           // SMTP 服务器端口
	Username string        // 登录账号
	Password string        // 登录密码或授权码
	From     string        // 发件人地址
	FromName string        // 发件人显示名
	TLS      bool          // 是否启用隐式 TLS
	Timeout  time.Duration // 发送超时
	Enabled  bool          // 是否启用邮件发送
}

// Validate 校验邮件通道配置的完整性。
func (c *SMTPConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.Host == "" {
		return fmt.Errorf("启用邮件发送时 SMTP 地址不能为空")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("SMTP 端口不合法: %d", c.Port)
	}
	if c.From == "" {
		return fmt.Errorf("启用邮件发送时发件人地址不能为空")
	}
	return nil
}
