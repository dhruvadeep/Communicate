package user

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID                     string
	Email                  string
	Username               string
	PasswordHash           string
	EmailVerifiedAt        *time.Time
	ProfileImageURL        *string
	ProfileImageObjectKey  *string
	CreatedAt              time.Time
	UpdatedAt              time.Time
	LastLoginAt            *time.Time
	IsActive               bool
	DeletedAt              *time.Time
}

func CreateUser(ctx context.Context, pool *pgxpool.Pool, email, username, passwordHash string, profileImageURL *string) (*User, error) {
	u := &User{}
	err := pool.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, profile_image_url)
		VALUES ($1, $2, $3, $4)
		RETURNING _id, email, username, password_hash, email_verified_at,
		          profile_image_url, profile_image_object_key,
		          created_at, updated_at, last_login_at, is_active, deleted_at
	`, email, username, passwordHash, profileImageURL).Scan(
		&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.EmailVerifiedAt,
		&u.ProfileImageURL, &u.ProfileImageObjectKey,
		&u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt,
		&u.IsActive, &u.DeletedAt,
	)
	if err != nil {
		log.Printf("create user: %v", err)
		return nil, err
	}
	return u, nil
}

func FindByID(ctx context.Context, pool *pgxpool.Pool, id string) (*User, error) {
	u := &User{}
	err := pool.QueryRow(ctx, `
		SELECT _id, email, username, password_hash, email_verified_at,
		       profile_image_url, profile_image_object_key,
		       created_at, updated_at, last_login_at, is_active, deleted_at
		FROM users
		WHERE _id = $1 AND deleted_at IS NULL
	`, id).Scan(
		&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.EmailVerifiedAt,
		&u.ProfileImageURL, &u.ProfileImageObjectKey,
		&u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt,
		&u.IsActive, &u.DeletedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		log.Printf("find user by id: %v", err)
		return nil, err
	}
	return u, nil
}

func UpdateProfileImage(ctx context.Context, pool *pgxpool.Pool, userID, url, objectKey string) error {
	_, err := pool.Exec(ctx, `
		UPDATE users SET profile_image_url = $2, profile_image_object_key = $3, updated_at = NOW()
		WHERE _id = $1
	`, userID, url, objectKey)
	if err != nil {
		log.Printf("update profile image: %v", err)
		return err
	}
	return nil
}

func SoftDelete(ctx context.Context, pool *pgxpool.Pool, userID string) error {
	_, err := pool.Exec(ctx, `
		UPDATE users SET deleted_at = NOW(), updated_at = NOW() WHERE _id = $1
	`, userID)
	if err != nil {
		log.Printf("soft delete user: %v", err)
		return err
	}
	return nil
}

func SetActive(ctx context.Context, pool *pgxpool.Pool, userID string, active bool) error {
	_, err := pool.Exec(ctx, `
		UPDATE users SET is_active = $2, updated_at = NOW() WHERE _id = $1
	`, userID, active)
	if err != nil {
		log.Printf("set active: %v", err)
		return err
	}
	return nil
}

func UpdatePassword(ctx context.Context, pool *pgxpool.Pool, userID, passwordHash string) error {
	_, err := pool.Exec(ctx, `
		UPDATE users SET password_hash = $2, updated_at = NOW() WHERE _id = $1
	`, userID, passwordHash)
	if err != nil {
		log.Printf("update password: %v", err)
		return err
	}
	return nil
}

func ExistsByEmail(ctx context.Context, pool *pgxpool.Pool, email string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL
		)
	`, email).Scan(&exists)
	if err != nil {
		log.Printf("exists by email: %v", err)
	}
	return exists, err
}

func FindByEmail(ctx context.Context, pool *pgxpool.Pool, email string) (*User, error) {
	u := &User{}
	err := pool.QueryRow(ctx, `
		SELECT _id, email, username, password_hash, email_verified_at,
		       profile_image_url, profile_image_object_key,
		       created_at, updated_at, last_login_at, is_active, deleted_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`, email).Scan(
		&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.EmailVerifiedAt,
		&u.ProfileImageURL, &u.ProfileImageObjectKey,
		&u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt,
		&u.IsActive, &u.DeletedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		log.Printf("find user by email: %v", err)
		return nil, err
	}
	return u, nil
}

func ExistsByUsername(ctx context.Context, pool *pgxpool.Pool, username string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM users WHERE username = $1 AND deleted_at IS NULL
		)
	`, username).Scan(&exists)
	if err != nil {
		log.Printf("exists by username: %v", err)
	}
	return exists, err
}
