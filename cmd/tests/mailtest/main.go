package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"Communicate/internal/config"
	"Communicate/internal/mail"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if cfg.SMTPHost == "" {
		fmt.Println("SKIP: SMTP_HOST is not set — mail not configured, nothing to test")
		return
	}

	if cfg.SMTPFrom == "" {
		fmt.Println("SKIP: SMTP_FROM is not set — mail not configured, nothing to test")
		return
	}

	port, err := strconv.Atoi(cfg.SMTPPort)
	if err != nil || port == 0 {
		port = 587
	}

	mailer := mail.New(mail.Config{
		Host:     cfg.SMTPHost,
		Port:     port,
		User:     cfg.SMTPUser,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
		FromName: cfg.SMTPFromName,
	})

	// MAIL_TEST_TO overrides the recipient(s), otherwise send to self.
	// Use comma to separate multiple addresses: a@x.com,b@x.com
	raw := strings.TrimSpace(os.Getenv("MAIL_TEST_TO"))
	if raw == "" {
		raw = cfg.SMTPFrom
	}

	var recipients []string
	for _, addr := range strings.Split(raw, ",") {
		addr = strings.TrimSpace(addr)
		if addr != "" {
			recipients = append(recipients, addr)
		}
	}

	fmt.Printf("sending test email to %v via %s:%d\n", recipients, cfg.SMTPHost, port)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	htmlBody := fmt.Sprintf(`
		<h1>Communicate — email test</h1>
		<p>This email confirms that the SMTP configuration is working correctly.</p>
		<ul>
			<li>Host: %s</li>
			<li>Time: %s</li>
		</ul>
		<p>If you received this, your mail setup is ready.</p>
	`, cfg.SMTPHost, time.Now().Format(time.RFC3339))

	textBody := fmt.Sprintf(
		"Communicate — email test\n\n"+
			"This email confirms that the SMTP configuration is working correctly.\n\n"+
			"Host: %s\nTime: %s\n\n"+
			"If you received this, your mail setup is ready.\n",
		cfg.SMTPHost, time.Now().Format(time.RFC3339),
	)

	if err := mailer.Send(ctx, "Communicate — email test", htmlBody, textBody, recipients...); err != nil {
		log.Fatalf("send: %v", err)
	}

	fmt.Println("email sent successfully")
}
