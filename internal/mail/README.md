# mail

SMTP email sender for Communicate. Supports HTML + plain text multipart emails.

## Quick start

```go
import (
    "context"
    "Communicate/internal/mail"
)

mailer := mail.New(mail.Config{
    Host:     "smtp.example.com",
    Port:     587,
    User:     "your-smtp-user",
    Password: "your-smtp-password",
    From:     "noreply@example.com",
    FromName: "Communicate",        // optional display name
})

ctx := context.Background()
err := mailer.Send(ctx,
    "user@example.com",            // to
    "Welcome to Communicate",      // subject
    "<h1>Welcome</h1><p>Hello!</p>", // htmlBody
    "Welcome. Hello!",             // textBody (optional)
)
```

## How it works

- **Both htmlBody + textBody** → `multipart/alternative` (text primary, HTML alternative)
- **Only htmlBody** → `text/html` single-part
- **Only textBody** → `text/plain` single-part
- SMTP auth is auto-detected (`SMTPAuthAuto` — PLAIN, LOGIN, etc.)
- TLS is mandatory (`TLSMandatory` with STARTTLS)
- For SSL (port 465) pass the port and the library handles implicit TLS

## Environment variables

```
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=your-smtp-user
SMTP_PASSWORD=your-smtp-password
SMTP_FROM=noreply@example.com
SMTP_FROM_NAME=Communicate
```

## Testing

```bash
make mail-test
```

This sends a test email to the address configured in `MAIL_TEST_TO` (falls back to `SMTP_FROM`).
