package user

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"Communicate/internal/handler"
	"Communicate/internal/handler/auth"
	"Communicate/internal/store/db/queries/user"
	"Communicate/internal/store/r2"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type uploadURLRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}

type uploadURLResponse struct {
	UploadURL string `json:"upload_url"`
	Key       string `json:"key"`
	PublicURL string `json:"public_url"`
}

var allowedMIME = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
}

// ProfileImageUploadURL returns a handler for POST /users/me/profile-image/upload-url.
func ProfileImageUploadURL(r2Client *r2.Client, publicURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req uploadURLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			handler.WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		if req.Filename == "" {
			handler.WriteError(w, http.StatusBadRequest, "filename is required")
			return
		}

		req.ContentType = strings.ToLower(strings.TrimSpace(req.ContentType))
		if !allowedMIME[req.ContentType] {
			handler.WriteError(w, http.StatusBadRequest, "content_type must be image/jpeg or image/png")
			return
		}

		key := fmt.Sprintf("profiles/%s_%s", uuid.New().String(), req.Filename)

		uploadURL, err := r2Client.PresignedUploadURL(key, req.ContentType, 15*time.Minute)
		if err != nil {
			log.Printf("generate presigned url: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "failed to generate upload URL")
			return
		}

		pubURL := ""
		if publicURL != "" {
			pubURL = publicURL + "/" + key
		}

		handler.WriteJSON(w, http.StatusOK, uploadURLResponse{
			UploadURL: uploadURL,
			Key:       key,
			PublicURL: pubURL,
		})
	}
}

type saveImageRequest struct {
	ObjectKey string `json:"object_key"`
}

// ProfileImageSave returns a handler for POST /users/me/profile-image.
// Called after the client uploads to R2, to save the key to the user record.
func ProfileImageSave(pool *pgxpool.Pool, publicURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserIDFromContext(r.Context())
		if userID == "" {
			handler.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var req saveImageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			handler.WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		if req.ObjectKey == "" {
			handler.WriteError(w, http.StatusBadRequest, "object_key is required")
			return
		}

		pubURL := ""
		if publicURL != "" {
			pubURL = publicURL + "/" + req.ObjectKey
		}

		if err := user.UpdateProfileImage(r.Context(), pool, userID, pubURL, req.ObjectKey); err != nil {
			log.Printf("save profile image: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		handler.WriteJSON(w, http.StatusOK, map[string]string{
			"profile_image_url": pubURL,
		})
	}
}
