package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
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
