package r2

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Client is a minimal Cloudflare R2 client using AWS Signature V4.
// No external dependencies — just stdlib crypto.
type Client struct {
	accessKey string
	secretKey string
	endpoint  string
	bucket    string
}

// New returns an R2 client for the given bucket.
func New(accessKey, secretKey, endpoint, bucket string) *Client {
	return &Client{
		accessKey: accessKey,
		secretKey: secretKey,
		endpoint:  strings.TrimRight(endpoint, "/"),
		bucket:    bucket,
	}
}

// PresignedUploadURL generates a presigned URL for a PUT upload.
// The client uses this URL to upload directly to R2 — your server never
// touches the file bytes. key is the object path (e.g. "profiles/abc_photo.jpg").
// contentType enforces what the client must send (e.g. "image/jpeg").
func (c *Client) PresignedUploadURL(key string, contentType string, expiry time.Duration) (string, error) {
	now := time.Now().UTC()
	timestamp := now.Format("20060102T150405Z")
	date := now.Format("20060102")
	region := "auto"
	service := "s3"

	credential := fmt.Sprintf("%s/%s/%s/%s/aws4_request", c.accessKey, date, region, service)

	signedHeaders := "content-type;host"

	query := url.Values{}
	query.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	query.Set("X-Amz-Credential", credential)
	query.Set("X-Amz-Date", timestamp)
	query.Set("X-Amz-Expires", fmt.Sprintf("%d", int(expiry.Seconds())))
	query.Set("X-Amz-SignedHeaders", signedHeaders)

	host := strings.TrimPrefix(strings.TrimPrefix(c.endpoint, "https://"), "http://")
	canonicalURI := "/" + c.bucket + "/" + key

	canonicalHeaders := "content-type:" + contentType + "\n" +
		"host:" + host + "\n"

	canonicalRequest := strings.Join([]string{
		"PUT",
		canonicalURI,
		query.Encode(),
		canonicalHeaders,
		signedHeaders,
		"UNSIGNED-PAYLOAD",
	}, "\n")

	algorithm := "AWS4-HMAC-SHA256"
	scope := fmt.Sprintf("%s/%s/%s/aws4_request", date, region, service)

	stringToSign := strings.Join([]string{
		algorithm,
		timestamp,
		scope,
		sha256Hex(canonicalRequest),
	}, "\n")

	signingKey := c.deriveSigningKey(date, region, service)
	signature := hmacHex(signingKey, stringToSign)

	query.Set("X-Amz-Signature", signature)

	return c.endpoint + canonicalURI + "?" + query.Encode(), nil
}

// deriveSigningKey builds the HMAC chain: key -> date -> region -> service -> aws4_request.
func (c *Client) deriveSigningKey(date, region, service string) []byte {
	kDate := hmacBytes([]byte("AWS4"+c.secretKey), date)
	kRegion := hmacBytes(kDate, region)
	kService := hmacBytes(kRegion, service)
	return hmacBytes(kService, "aws4_request")
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func hmacHex(key []byte, data string) string {
	return hex.EncodeToString(hmacBytes(key, data))
}

func hmacBytes(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}
