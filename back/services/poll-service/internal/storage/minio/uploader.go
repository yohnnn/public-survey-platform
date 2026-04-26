package minio

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/models"
)

type Config struct {
	Endpoint     string
	AccessKey    string
	SecretKey    string
	Bucket       string
	UseSSL       bool
	PublicBase   string
	PresignTTL   time.Duration
	MaxFileBytes int64
}

type Uploader struct {
	client       *minio.Client
	bucket       string
	publicBase   string
	presignTTL   time.Duration
	maxFileBytes int64
}

func NewUploader(cfg Config) (*Uploader, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	accessKey := strings.TrimSpace(cfg.AccessKey)
	secretKey := strings.TrimSpace(cfg.SecretKey)
	bucket := strings.TrimSpace(cfg.Bucket)
	publicBase := strings.TrimRight(strings.TrimSpace(cfg.PublicBase), "/")

	if endpoint == "" {
		return nil, fmt.Errorf("minio endpoint is required")
	}
	if accessKey == "" {
		return nil, fmt.Errorf("minio access key is required")
	}
	if secretKey == "" {
		return nil, fmt.Errorf("minio secret key is required")
	}
	if bucket == "" {
		return nil, fmt.Errorf("minio bucket is required")
	}
	if publicBase == "" {
		return nil, fmt.Errorf("minio public base url is required")
	}
	if cfg.PresignTTL <= 0 {
		return nil, fmt.Errorf("minio presign ttl must be > 0")
	}
	if cfg.MaxFileBytes <= 0 {
		return nil, fmt.Errorf("minio max file bytes must be > 0")
	}

	if _, err := url.ParseRequestURI(publicBase); err != nil {
		return nil, fmt.Errorf("invalid minio public base url: %w", err)
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &Uploader{
		client:       client,
		bucket:       bucket,
		publicBase:   publicBase,
		presignTTL:   cfg.PresignTTL,
		maxFileBytes: cfg.MaxFileBytes,
	}, nil
}

func (u *Uploader) CreatePollImageUploadURL(ctx context.Context, userID, fileName, contentType string, sizeBytes int64) (models.PollImageUpload, error) {
	if strings.TrimSpace(userID) == "" {
		return models.PollImageUpload{}, models.ErrUnauthorized
	}
	if sizeBytes <= 0 || sizeBytes > u.maxFileBytes {
		return models.PollImageUpload{}, models.ErrImageTooLarge
	}

	normalizedType := normalizeContentType(contentType)
	if _, ok := allowedImageContentTypes[normalizedType]; !ok {
		return models.PollImageUpload{}, models.ErrUnsupportedMime
	}

	objectKey := buildObjectKey(userID, fileName, normalizedType)
	uploadURL, err := u.client.PresignedPutObject(ctx, u.bucket, objectKey, u.presignTTL)
	if err != nil {
		return models.PollImageUpload{}, err
	}

	imageURL, err := joinURLPath(u.publicBase, objectKey)
	if err != nil {
		return models.PollImageUpload{}, models.ErrInvalidArgument
	}

	expiresInSeconds := int64(u.presignTTL.Seconds())
	if expiresInSeconds <= 0 {
		expiresInSeconds = 1
	}

	return models.PollImageUpload{
		ObjectKey:        objectKey,
		UploadURL:        uploadURL.String(),
		ImageURL:         imageURL,
		ExpiresInSeconds: expiresInSeconds,
	}, nil
}

var allowedImageContentTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/jpg":  {},
	"image/png":  {},
	"image/webp": {},
	"image/gif":  {},
}

func normalizeContentType(contentType string) string {
	v := strings.ToLower(strings.TrimSpace(contentType))
	if i := strings.Index(v, ";"); i >= 0 {
		v = strings.TrimSpace(v[:i])
	}
	return v
}

func extensionForUpload(fileName, contentType string) string {
	if ext := strings.ToLower(strings.TrimSpace(filepath.Ext(fileName))); ext != "" {
		return ext
	}

	switch contentType {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ""
	}
}

func buildObjectKey(userID, fileName, contentType string) string {
	ext := extensionForUpload(fileName, contentType)
	if ext == "" {
		ext = ".bin"
	}

	return path.Join(
		"polls",
		sanitizePathPart(userID),
		fmt.Sprintf("%d-%s%s", time.Now().UTC().Unix(), randomHex(8), ext),
	)
}

func sanitizePathPart(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return "unknown"
	}

	b := strings.Builder{}
	for _, r := range v {
		isLower := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if isLower || isDigit || r == '-' || r == '_' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('-')
	}

	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "unknown"
	}
	return out
}

func randomHex(byteLen int) string {
	if byteLen <= 0 {
		byteLen = 8
	}
	buf := make([]byte, byteLen)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func joinURLPath(base, objectKey string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	parsed.Path = path.Join(parsed.Path, objectKey)
	return parsed.String(), nil
}
