package main

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	err = r.ParseMultipartForm(maxMemory)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to parse form", err)
		return
	}

	fileData, imageHeader, err := r.FormFile("thumbnail")

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to extract image data", err)
		return
	}
	defer fileData.Close()

	mediaType, _, err := mime.ParseMediaType(imageHeader.Header.Get("Content-Type"))
	if err != nil {

		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type", err)
		return
	}

	if mediaType != "image/jpeg" && mediaType != "image/png" {

		respondWithError(w, http.StatusBadRequest, "Invalid file type", err)
		return
	}
	err = verifyMediaType(mediaType)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid media type", err)
		return
	}

	thumbnailData, err := io.ReadAll(fileData)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to extract image data", err)
		return
	}

	videoMetaData, err := cfg.db.GetVideo(videoID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to load video", err)
		return
	}

	if videoMetaData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	fileName, err := generateFileName()

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate filename", err)
		return

	}
	assetPath := getAssetPath(fileName, mediaType)
	assetDiskPath := cfg.getAssetDiskPath(assetPath)

	file, err := os.Create(assetDiskPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create file", err)
		return
	}
	defer file.Close()

	_, err = io.Copy(file, bytes.NewReader(thumbnailData))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to write to disk", err)
		return
	}

	thumbnailURL := cfg.getAssetURL(assetPath)

	err = cfg.db.UpdateVideo(database.Video{
		ID:           videoMetaData.ID,
		ThumbnailURL: &thumbnailURL,
		CreateVideoParams: database.CreateVideoParams{
			Title:       videoMetaData.Title,
			Description: videoMetaData.Description,
			UserID:      videoMetaData.UserID,
		},
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating video", err)
		return
	}

	updatedVideo, err := cfg.db.GetVideo(videoID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error getting updated video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, updatedVideo)
}
