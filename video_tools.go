package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	data := bytes.Buffer{}
	cmd.Stdout = &data
	errBuf := bytes.Buffer{}
	cmd.Stderr = &errBuf

	err := cmd.Run()

	if err != nil {
		return "", fmt.Errorf("failed to run ffprobe: %s, stderr: %s", err, errBuf.String())
	}

	type response struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}
	resp := response{}
	err = json.Unmarshal(data.Bytes(), &resp)

	if err != nil {
		return "", fmt.Errorf("error unmarshaling data: %s", err)
	}

	height, width := simplifyAspectRatio(resp.Streams[0].Height, resp.Streams[0].Width)

	ratio := float64(width) / float64(height)

	const tolerance = 0.1
	if math.Abs(ratio-16.0/9.0) <= tolerance {
		return "16:9", nil
	} else if math.Abs(ratio-9.0/16.0) <= tolerance {
		return "9:16", nil
	}
	return "other", nil
}

func gcd(a, b int) int {
	if b == 0 {
		return a
	}
	return gcd(b, a%b)
}

func simplifyAspectRatio(width, height int) (int, int) {
	divisor := gcd(width, height)
	return width / divisor, height / divisor
}

func processVideoForFastStart(filePath string) (string, error) {
	outputFile := filePath + ".processing"

	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputFile)
	data := bytes.Buffer{}
	cmd.Stdout = &data
	errBuf := bytes.Buffer{}
	cmd.Stderr = &errBuf

	err := cmd.Run()

	if err != nil {
		return "", fmt.Errorf("failed to run ffmpeg: %s, stderr: %s", err, errBuf.String())
	}

	return outputFile, nil
}

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)

	req, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}, s3.WithPresignExpires(expireTime))

	if err != nil {
		return "", fmt.Errorf("unable to generate presigned URL: %s", err)
	}

	url := req.URL

	return url, nil

}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil || *video.VideoURL == "" {
		// If the video is a draft, we can just return it unmodified
		return video, nil
	}

	// Split the "bucket,key" format
	parts := strings.Split(*video.VideoURL, ",")
	if len(parts) != 2 {
		return database.Video{}, fmt.Errorf("invalid VideoURL format for video with ID %s", video.ID)
	}

	bucket, key := parts[0], parts[1]

	// Generate the presigned URL
	presignedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, 10*time.Minute)
	if err != nil {
		return database.Video{}, fmt.Errorf("error generating presigned URL for video ID %s: %v", video.ID, err)
	}

	// Set the presigned URL as the updated VideoURL
	video.VideoURL = &presignedURL
	return video, nil
}
