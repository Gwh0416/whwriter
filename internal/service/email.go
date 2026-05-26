package service

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net/smtp"

	"whwriter/internal/config"
)

type EmailService struct {
	cfg config.SMTPConfig
}

func NewEmailService(cfg config.SMTPConfig) *EmailService {
	return &EmailService{cfg: cfg}
}

func (s *EmailService) GenerateCode() string {
	code := ""
	for i := 0; i < 6; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(10))
		code += fmt.Sprintf("%d", n)
	}
	return code
}

func (s *EmailService) SendVerificationCode(to, code string) error {
	if s.cfg.Host == "" {
		log.Printf("[EMAIL] SMTP not configured, code for %s: %s", to, code)
		return nil
	}

	subject := "文豪写作 - 邮箱验证码"
	body := fmt.Sprintf(`
<html>
<body style="font-family: sans-serif; padding: 20px;">
  <h2 style="color: #f5af19;">文豪写作</h2>
  <p>您的验证码是：</p>
  <h1 style="color: #f12711; letter-spacing: 4px;">%s</h1>
  <p>验证码 10 分钟内有效，请勿泄露给他人。</p>
</body>
</html>`, code)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		s.cfg.From, to, subject, body)

	addr := fmt.Sprintf("%s:%s", s.cfg.Host, s.cfg.Port)
	auth := smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)

	if err := smtp.SendMail(addr, auth, s.cfg.From, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	log.Printf("[EMAIL] verification code sent to %s", to)
	return nil
}
