// Package mailer 提供账户与通知邮件的统一发送入口，未配置 SMTP 时退化为日志输出。
package mailer

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"api/internal/platform/config"
)

// Mailer 抽象邮件发送能力，便于在未配置 SMTP 时替换为日志实现。
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// SMTPMailer 通过 SMTP 发送纯文本邮件。
type SMTPMailer struct {
	cfg config.SMTPConfig // SMTP 连接参数
}

// NewSMTPMailer 创建 SMTP 邮件发送器。
func NewSMTPMailer(cfg config.SMTPConfig) *SMTPMailer {
	return &SMTPMailer{cfg: cfg}
}

// Send 发送纯文本邮件，主题与正文中的换行会被清理，避免头注入。
func (m *SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	host := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	fromName := m.cfg.FromName
	if fromName == "" {
		fromName = "Veristack"
	}
	message := fmt.Sprintf(
		"From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n",
		sanitizeHeader(fromName), m.cfg.From, sanitizeHeader(to), sanitizeHeader(subject), body,
	)
	auth := smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	return smtp.SendMail(host, auth, m.cfg.From, []string{to}, []byte(message))
}

// LogMailer 在未配置邮件通道时把邮件写入标准输出，仅用于本地开发。
type LogMailer struct{}

// Send 输出邮件摘要，便于本地联调。
func (LogMailer) Send(_ context.Context, to, subject, body string) error {
	fmt.Printf("邮件已发送至 %s：%s\n%s\n", to, subject, body)
	return nil
}

// sanitizeHeader 去除邮件头中的换行，防止头注入。
func sanitizeHeader(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "\r", ""), "\n", "")
}
