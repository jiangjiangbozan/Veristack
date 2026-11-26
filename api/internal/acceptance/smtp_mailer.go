package acceptance

import (
	"context"
	"fmt"
	"net/smtp"
	"time"

	"api/internal/platform/config"
)

// SMTPMailer 通过 SMTP 发送客户验证邮件。
type SMTPMailer struct {
	cfg config.SMTPConfig // SMTP 连接参数
}

// NewSMTPMailer 创建 SMTP 邮件发送器。
func NewSMTPMailer(cfg config.SMTPConfig) *SMTPMailer {
	return &SMTPMailer{cfg: cfg}
}

// SendVerificationCode 发送验证码邮件。
func (m *SMTPMailer) SendVerificationCode(_ context.Context, email, code string) error {
	host := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: Veristack 验收验证码\r\n\r\n您的验收验证码是 %s，15 分钟内有效。\r\n", m.cfg.From, email, code)
	auth := smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	timeout := m.cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return smtp.SendMail(host, auth, m.cfg.From, []string{email}, []byte(message))
}
