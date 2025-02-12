package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func getAssetPath(videoID string, mediaType string) string {
	ext := mediaTypeToExt(mediaType)
	return fmt.Sprintf("%s%s", videoID, ext)
}

func (cfg apiConfig) getAssetDiskPath(assetPath string) string {
	return filepath.Join(cfg.assetsRoot, assetPath)
}

func (cfg apiConfig) getAssetURL(assetPath string) string {

	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetPath)
}

func mediaTypeToExt(mediaType string) string {
	parts := strings.Split(mediaType, "/")
	if len(parts) != 2 {
		return ".bin"
	}
	return "." + parts[1]
}

func generateFileName() (string, error) {
	data := make([]byte, 32)

	_, err := rand.Read(data)

	if err != nil {
		return "", fmt.Errorf("failed to generate data: %s", err)
	}
	encoded := base64.URLEncoding.EncodeToString(data)

	return encoded, nil

}

func verifyMediaType(mediaType string) error {
	mt, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		return fmt.Errorf("failed to parse media type: %s", err)
	}

	if mt != "image/jpeg" && mt != "image/png" {
		return fmt.Errorf("invalid media type: %s", mt)
	}
	return nil
}

func (cfg apiConfig) generateBucketURL() string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/", cfg.s3Bucket, cfg.s3Region)
}
