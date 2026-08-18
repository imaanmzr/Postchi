package email

import (
	"fmt"
	"net/smtp"
	"strings"

	"github.com/imaanmzr/postchi/backend/internal/shared/config"
)

type Sender struct {
	cfg *config.Config
}

func NewSender(cfg *config.Config) *Sender {
	return &Sender{cfg: cfg}
}

func (s *Sender) Send(to, subject, body string) error {
	if !s.cfg.SMTPConfigured() {
		return fmt.Errorf("SMTP is not configured (set SMTP_HOST)")
	}
	from := s.cfg.SMTPFrom
	msg := strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")
	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)
	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPass, s.cfg.SMTPHost)
	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
}

func (s *Sender) SendInvite(to, workspaceName, inviteURL string) error {
	subject := fmt.Sprintf("You're invited to %s on Postchi", workspaceName)
	body := fmt.Sprintf("You've been invited to join workspace \"%s\" on Postchi.\n\nAccept your invite:\n%s\n\nThis link expires in %d hours.\n",
		workspaceName, inviteURL, int(s.cfg.InviteTTL.Hours()))
	return s.Send(to, subject, body)
}
