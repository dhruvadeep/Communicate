# API Responses

Base URL: `http://localhost:8080`

All responses are JSON. Errors have the shape `{"error":"message"}` unless noted otherwise.

---

## Public endpoints

### `GET /health`

Health check. No auth required.

**200** — server and database are up.
```json
{
  "status": "healthy",
  "db": "up",
  "db_ping_ms": 12.34,
  "uptime": "1h 23m 45s",
  "uptime_seconds": 5025.12,
  "timestamp": "2026-07-03T12:00:00Z"
}
```

**503** — database is unreachable.
```json
{
  "status": "unhealthy",
  "db": "down",
  "timestamp": "2026-07-03T12:00:00Z"
}
```

---

### `POST /auth/register`

Create a new account. Rate-limited (default: 5 attempts per 15 min).

Request:
```json
{
  "email": "user@example.com",
  "username": "johndoe",
  "password": "password123",
  "profile_image_url": null
}
```

**201** — account created. A verification email is sent.
```json
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "username": "johndoe",
    "profile_image_url": null,
    "created_at": "2026-07-03T12:00:00Z",
    "updated_at": "2026-07-03T12:00:00Z"
  }
}
```

**400** — validation failure.
```json
{"error": "valid email is required"}
```
```json
{"error": "username must be between 3 and 50 characters"}
```
```json
{"error": "password must be at least 8 characters"}
```
```json
{"error": "invalid email syntax"}
```
```json
{"error": "disposable email addresses are not allowed"}
```
```json
{"error": "domain \"example.com\" has no MX records — cannot receive email"}
```
```json
{"error": "domain \"bogus.xyz\" does not exist"}
```

**409** — duplicate.
```json
{"error": "email already taken"}
```
```json
{"error": "username already taken"}
```
```json
{"error": "email or username already taken"}
```

**429** — rate limited.
```json
{
  "error": "too many attempts",
  "locked_until": "2026-07-03T12:30:00Z"
}
```

**500** — internal server error.
```json
{"error": "internal server error"}
```

---

### `GET /auth/verify-email?token=<token>`

Verify email address. Link is sent after registration.

**200** — verified successfully.
```json
{"message": "email verified successfully"}
```

**400** — token missing, invalid, or expired.
```json
{"error": "token is required"}
```
```json
{"error": "invalid or expired token"}
```

**500** — internal server error.
```json
{"error": "internal server error"}
```

---

### `POST /auth/resend-verification`

Request a new verification email. Does not reveal whether the email is registered.

Request:
```json
{"email": "user@example.com"}
```

**200** — always returned (to avoid leaking registered emails).
```json
{"message": "if that email is registered, a verification email has been sent"}
```
```json
{"message": "verification email sent"}
```
```json
{"message": "email is already verified"}
```

**400** — bad request.
```json
{"error": "valid email is required"}
```

**500** — internal server error.
```json
{"error": "internal server error"}
```

---

### `POST /auth/login`

Authenticate and receive tokens. Rate-limited (default: 5 attempts per 15 min).  
Requires email to be verified.

Request:
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**200** — login successful.
```json
{
  "access_token": "hex-64-chars",
  "refresh_token": "hex-64-chars",
  "expires_at": "2026-07-03T12:15:00Z",
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "username": "johndoe",
    "profile_image_url": null,
    "email_verified": true,
    "created_at": "2026-07-03T12:00:00Z"
  }
}
```

**400** — validation failure.
```json
{"error": "email and password are required"}
```

**401** — invalid credentials.
```json
{"error": "invalid email or password"}
```
```json
{"error": "account is disabled"}
```

**403** — email not verified.
```json
{"error": "email not verified — please check your inbox"}
```

**429** — rate limited.
```json
{
  "error": "too many attempts",
  "locked_until": "2026-07-03T12:30:00Z"
}
```

**500** — internal server error.
```json
{"error": "internal server error"}
```

---

### `POST /auth/refresh`

Rotate tokens. Old session is revoked, new one created.

Request:
```json
{"refresh_token": "hex-64-chars"}
```

**200** — new token pair.
```json
{
  "access_token": "hex-64-chars",
  "refresh_token": "hex-64-chars",
  "expires_at": "2026-07-03T12:30:00Z"
}
```

**400** — validation failure.
```json
{"error": "refresh_token is required"}
```

**401** — invalid or expired.
```json
{"error": "invalid or expired refresh token"}
```

**500** — internal server error.
```json
{"error": "internal server error"}
```

---

### `POST /auth/forgot-password`

Request a password reset link. Rate-limited. Does not reveal whether the email is registered.

Request:
```json
{"email": "user@example.com"}
```

**200** — always returned.
```json
{"message": "if that email is registered, a reset link has been sent"}
```

**500** — internal server error.
```json
{"error": "internal server error"}
```

---

### `POST /auth/reset-password`

Set a new password using a token from the reset email. Rate-limited.  
Revokes ALL sessions on success — user must log in again.

Token can be sent in the JSON body or as a query parameter.

Request (JSON body):
```json
{
  "token": "hex-64-chars",
  "password": "newpassword123"
}
```

Request (query param):
```
POST /auth/reset-password?token=hex-64-chars
{"password": "newpassword123"}
```

**200** — password changed.
```json
{"message": "password reset successfully — all sessions have been revoked, please log in again"}
```

**400** — validation failure.
```json
{"error": "token is required"}
```
```json
{"error": "password must be at least 8 characters"}
```
```json
{"error": "invalid or expired token"}
```

**429** — rate limited.
```json
{
  "error": "too many attempts",
  "locked_until": "2026-07-03T12:30:00Z"
}
```

**500** — internal server error.
```json
{"error": "internal server error"}
```

---

## Protected endpoints

All require header: `Authorization: Bearer <access_token>`

---

### `POST /auth/logout`

Revoke ALL sessions for the authenticated user. Logs out from every device.

No request body.

**200** — logged out.
```json
{"message": "logged out from all devices"}
```

**401** — missing or invalid token.
```json
{"error": "missing authorization header"}
```
```json
{"error": "invalid or expired token"}
```
```json
{"error": "unauthorized"}
```

**500** — internal server error.
```json
{"error": "internal server error"}
```

---

### `GET /auth/session`

Check if the current access token is still valid. Returns user info if yes.  
Used by the frontend to hydrate state without re-login.

No request body.

**200** — session valid.
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "username": "johndoe",
  "profile_image_url": null,
  "email_verified": true,
  "created_at": "2026-07-03T12:00:00Z"
}
```

**401** — token invalid, expired, revoked, or user disabled/deleted.
```json
{"error": "missing authorization header"}
```
```json
{"error": "invalid or expired token"}
```
```json
{"error": "unauthorized"}
```

**500** — internal server error.
```json
{"error": "internal server error"}
```

---

### `GET /users/me`

Get the authenticated user's profile.

No request body.

**200** — profile.
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "username": "johndoe",
  "profile_image_url": "https://images.example.com/profiles/abc_photo.jpg",
  "email_verified": true,
  "created_at": "2026-07-03T12:00:00Z"
}
```

**401** — token invalid.
```json
{"error": "missing authorization header"}
```
```json
{"error": "invalid or expired token"}
```
```json
{"error": "unauthorized"}
```

**404** — user not found (soft-deleted).
```json
{"error": "user not found"}
```

**500** — internal server error.
```json
{"error": "internal server error"}
```

---

### `POST /users/me/password`

Change password while logged in. Must provide current password.  
Revokes ALL sessions on success — must log in again.

Request:
```json
{
  "current_password": "oldpassword",
  "new_password": "newpassword123"
}
```

**200** — password changed.
```json
{"message": "password changed — all sessions revoked, please log in again"}
```

**400** — validation.
```json
{"error": "new password must be at least 8 characters"}
```

**401** — current password is wrong.
```json
{"error": "missing authorization header"}
```
```json
{"error": "invalid or expired token"}
```
```json
{"error": "current password is incorrect"}
```
```json
{"error": "unauthorized"}
```

**404** — user not found.
```json
{"error": "user not found"}
```

**500** — internal server error.
```json
{"error": "internal server error"}
```

---

### `DELETE /users/me`

Soft-delete the account (sets `deleted_at`, does not remove the row).  
Revokes ALL sessions.

No request body.

**200** — deleted.
```json
{"message": "account deleted"}
```

**401** — token invalid.
```json
{"error": "missing authorization header"}
```
```json
{"error": "invalid or expired token"}
```
```json
{"error": "unauthorized"}
```

**500** — internal server error.
```json
{"error": "internal server error"}
```

---

### `POST /users/me/deactivate`

Toggle `is_active` on/off. Deactivating revokes all sessions.

No request body.

**200** — state toggled.
```json
{
  "message": "account deactivated — all sessions revoked",
  "is_active": false
}
```
```json
{
  "message": "account activated",
  "is_active": true
}
```

**401** — token invalid.
```json
{"error": "missing authorization header"}
```
```json
{"error": "invalid or expired token"}
```
```json
{"error": "unauthorized"}
```

**404** — user not found.
```json
{"error": "user not found"}
```

**500** — internal server error.
```json
{"error": "internal server error"}
```

---

### `POST /users/me/profile-image/upload-url`

Get a presigned R2 upload URL. The client uploads directly to this URL, then calls `/users/me/profile-image` to save the key.

Request:
```json
{
  "filename": "photo.jpg",
  "content_type": "image/jpeg"
}
```

**200** — presigned URL ready.
```json
{
  "upload_url": "https://bucket.r2.cloudflarestorage.com/communicate/profiles/uuid_photo.jpg?X-Amz-...",
  "key": "profiles/uuid_photo.jpg",
  "public_url": "https://images.example.com/profiles/uuid_photo.jpg"
}
```

**400** — validation.
```json
{"error": "filename is required"}
```
```json
{"error": "content_type must be image/jpeg or image/png"}
```

**401** — token invalid.
```json
{"error": "missing authorization header"}
```
```json
{"error": "invalid or expired token"}
```
```json
{"error": "unauthorized"}
```

**500** — internal server error.
```json
{"error": "failed to generate upload URL"}
```

---

### `POST /users/me/profile-image`

Save the uploaded image key to the user profile. Call AFTER uploading to the presigned URL.

Request:
```json
{"object_key": "profiles/uuid_photo.jpg"}
```

**200** — saved.
```json
{"profile_image_url": "https://images.example.com/profiles/uuid_photo.jpg"}
```

**400** — validation.
```json
{"error": "object_key is required"}
```

**401** — token invalid.
```json
{"error": "missing authorization header"}
```
```json
{"error": "invalid or expired token"}
```
```json
{"error": "unauthorized"}
```

**500** — internal server error.
```json
{"error": "internal server error"}
```

---

## Rate limiting

These endpoints are rate-limited per IP:

| Endpoint | Identifier |
|---|---|
| `POST /auth/register` | `ip:register` |
| `POST /auth/login` | `ip:login` |
| `POST /auth/forgot-password` | `ip:forgot-password` |
| `POST /auth/reset-password` | `ip:reset-password` |

**Default:** 5 failed attempts in a 15-minute window → 30-minute lock.  
Configure via `RATE_LIMIT_MAX_ATTEMPTS`, `RATE_LIMIT_WINDOW`, `RATE_LIMIT_LOCK_DURATION`.

Successful requests clear the lock. Only failures (4xx/5xx) count toward the limit.

When locked, all responses are:

**429** — too many requests.
```json
{
  "error": "too many attempts",
  "locked_until": "2026-07-03T12:30:00Z"
}
```

Response includes `X-RateLimit-Remaining` header on failed requests (when not yet locked).

---

## Common error patterns

| Status | Meaning |
|---|---|
| 400 | Client sent something invalid (missing fields, bad values) |
| 401 | Missing or invalid `Authorization` header |
| 403 | Authenticated but not allowed (e.g. email not verified for login) |
| 404 | Resource not found (soft-deleted user, unknown route) |
| 409 | Conflict (duplicate email/username) |
| 429 | Rate limited — wait until `locked_until` |
| 500 | Server error — check logs |
| 503 | Service unavailable (health check: DB down) |
