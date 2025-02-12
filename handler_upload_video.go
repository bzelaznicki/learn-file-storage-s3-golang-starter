package main

import (
	"io"
	"log"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxVideo)

	videoIdString := r.PathValue("videoID")

	videoID, err := uuid.Parse(videoIdString)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid video ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't get video", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusForbidden, "You can't upload this video", err)
		return
	}

	file, fileHeader, err := r.FormFile("video")

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to process video", err)
		return
	}

	mt, _, err := mime.ParseMediaType(fileHeader.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type", err)
		return
	}

	if mt != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", err)
		return
	}
	tempFileName := "tubely-vid.mp4"
	tempFile, err := os.CreateTemp("", tempFileName)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to generate temporary file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to copy file data", err)
		return
	}

	aspectRatio, err := getVideoAspectRatio(tempFile.Name())

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to calculate aspect ratio", err)
		return
	}
	var prefix string
	switch aspectRatio {
	case "16:9":
		prefix = "landscape"
	case "9:16":
		prefix = "portrait"
	default:
		prefix = "other"
	}

	_, err = tempFile.Seek(0, io.SeekStart)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to reset file", err)
		return
	}

	processedFile, err := processVideoForFastStart(tempFile.Name())

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to process file", err)
		return
	}
	defer os.Remove(processedFile)

	procFile, err := os.Open(processedFile)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to open temporary file", err)
		return
	}

	defer procFile.Close()

	fileName, err := generateFileName()

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate filename", err)
		return
	}

	fileNameWithPath := prefix + "/" + fileName

	fullName := getAssetPath(fileNameWithPath, mt)

	s3PutObject := s3.PutObjectInput{
		Bucket:      aws.String(cfg.s3Bucket),
		Key:         aws.String(fullName),
		Body:        procFile,
		ContentType: aws.String("video/mp4"),
	}

	log.Printf("Uploading to bucket: %s, key: %s, region: %s", cfg.s3Bucket, fullName, cfg.s3Region)
	_, err = cfg.s3Client.PutObject(r.Context(), &s3PutObject)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error uploading video", err)
		return
	}
	bucketURL := cfg.generateBucketURL()
	s3URL := bucketURL + fullName

	err = cfg.db.UpdateVideo(database.Video{
		ID:           videoID,
		ThumbnailURL: video.ThumbnailURL,
		VideoURL:     &s3URL,
		CreateVideoParams: database.CreateVideoParams{
			Title:       video.Title,
			Description: video.Description,
			UserID:      video.UserID,
		},
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update video", err)
		return
	}

	updatedVideo, err := cfg.db.GetVideo(videoID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error getting updated video", err)
		return
	}
	respondWithJSON(w, http.StatusOK, updatedVideo)
}
