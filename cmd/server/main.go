package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"Communicate/internal/comm"
	"Communicate/internal/config"
	"Communicate/internal/handler"
	"Communicate/internal/handler/auth"
	"Communicate/internal/handler/signaling"
	handleruser "Communicate/internal/handler/user"
	"Communicate/internal/mail"
	"Communicate/internal/store/db"
	"Communicate/internal/store/db/migrate"
	"Communicate/internal/store/db/models/indexes"
	"Communicate/internal/store/db/models/tables"
	"Communicate/internal/store/db/queries/sessions"
	"Communicate/internal/store/db/queries/user"
	"Communicate/internal/store/r2"
	sqlitestore "Communicate/internal/store/sqlite"
	"Communicate/internal/store/sqlite/queries/ratelimit"
	"Communicate/internal/verify"
)

func main() {
	ctx := context.Background()

	// ── PostgreSQL ────────────────────────────────────────────────────
	database, err := db.Open(ctx)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	if err := migrate.RunMigrations(ctx, database.Pool(), tables.All...); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	if err := migrate.RunIndexes(ctx, database.Pool(), indexes.All...); err != nil {
		log.Fatalf("failed to run indexes: %v", err)
	}

	log.Println("database connected and migrations applied")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	pool := database.Pool()
	serverStart := time.Now()

	// ── SQLite (rate limiting) ────────────────────────────────────────
	rateDB, err := sqlitestore.Open()
	if err != nil {
		log.Fatalf("failed to open sqlite: %v", err)
	}
	defer sqlitestore.Close()

	if err := sqlitestore.RunMigrations(rateDB); err != nil {
		log.Fatalf("failed to run sqlite migrations: %v", err)
	}

	ratelimit.Configure(cfg.RateLimitMaxAttempts, cfg.RateLimitWindow, cfg.RateLimitLockDuration)

	// Periodic cleanup of old rate limit entries.
	go func() {
		for {
			time.Sleep(1 * time.Hour)
			ratelimit.Cleanup(rateDB)
		}
	}()

	// ── Hub (WebSocket signalling) ────────────────────────────────────
	hub := comm.NewHub()

	// Enable mid-session auth for UDP peers. When a UDP client sends a
	// MSG with {"token":"<access_token>",...}, the hub validates it
	// and sets the peer's display name + avatar from the DB.
	hub.OnAuth = func(token string) (string, string, bool) {
		s, err := sessions.GetByAccessToken(ctx, pool, token)
		if err != nil || s == nil {
			return "", "", false
		}
		u, err := user.FindByID(ctx, pool, s.UserID)
		if err != nil || u == nil {
			return "", "", false
		}
		avatar := ""
		if u.ProfileImageURL != nil {
			avatar = *u.ProfileImageURL
		}
		return u.Username, avatar, true
	}

	go hub.Run()

	// ── UDP listener (JavaFX / non-WebSocket clients) ─────────────────
	go func() {
		log.Printf("UDP listener opening on :%s ...", cfg.UDPPort)
		if err := comm.ListenUDP("localhost:"+cfg.UDPPort, hub); err != nil {
			log.Fatalf("failed to open UDP port :%s: %v", cfg.UDPPort, err)
		}
	}()

	// ── Services ──────────────────────────────────────────────────────
	r2Client := r2.New(
		cfg.R2AccessKeyID, cfg.R2SecretAccessKey,
		cfg.R2Endpoint, cfg.R2BucketName,
	)

	emailVerifier := verify.New(verify.Method(cfg.EmailVerifierMethod))

	smtpPort, err := strconv.Atoi(cfg.SMTPPort)
	if err != nil || smtpPort == 0 {
		smtpPort = 587
	}
	mailer := mail.New(mail.Config{
		Host:     cfg.SMTPHost,
		Port:     smtpPort,
		User:     cfg.SMTPUser,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
		FromName: cfg.SMTPFromName,
	})

	// ── Public routes ─────────────────────────────────────────────────
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.Health(pool, serverStart))
	mux.HandleFunc("GET /ws", signaling.WebSocket(hub, pool))
	mux.HandleFunc("POST /auth/register", handler.RateLimit(rateDB, "register",
		auth.Register(pool, emailVerifier, mailer, cfg.BaseURL)))
	mux.HandleFunc("GET /auth/verify-email", auth.VerifyEmail(pool))
	mux.HandleFunc("POST /auth/resend-verification", auth.ResendVerification(pool, mailer, cfg.BaseURL))
	mux.HandleFunc("POST /auth/login", handler.RateLimit(rateDB, "login",
		auth.Login(pool)))
	mux.HandleFunc("POST /auth/refresh", auth.Refresh(pool))
	mux.HandleFunc("POST /auth/forgot-password", handler.RateLimit(rateDB, "forgot-password",
		auth.ForgotPassword(pool, mailer, cfg.BaseURL)))
	mux.HandleFunc("GET /auth/reset-password", auth.ResetPassword(pool))
	mux.HandleFunc("POST /auth/reset-password", handler.RateLimit(rateDB, "reset-password",
		auth.ResetPassword(pool)))

	// ── Protected routes ──────────────────────────────────────────────
	protected := http.NewServeMux()
	protected.HandleFunc("POST /auth/logout", auth.Logout(pool))
	protected.HandleFunc("GET /auth/session", auth.Session(pool))
	protected.HandleFunc("GET /users/me", handleruser.Me(pool))
	protected.HandleFunc("POST /users/me/password", handleruser.ChangePassword(pool))
	protected.HandleFunc("DELETE /users/me", handleruser.DeleteAccount(pool))
	protected.HandleFunc("POST /users/me/deactivate", handleruser.Deactivate(pool))
	protected.HandleFunc("POST /users/me/profile-image/upload-url", handleruser.ProfileImageUploadURL(r2Client, cfg.R2PublicURL))
	protected.HandleFunc("POST /users/me/profile-image", handleruser.ProfileImageSave(pool, cfg.R2PublicURL))

	mux.Handle("/", auth.RequireAuth(pool)(protected))

	addr := "localhost:" + cfg.ServerPort
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, handler.Logger(handler.CORS(mux))); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
