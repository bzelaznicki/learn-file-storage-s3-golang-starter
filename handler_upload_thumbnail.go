package main

import (
	"fmt"
	"io"
	"net/http"

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

	mediaType := imageHeader.Header.Get("Content-Type")

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

	tnail := thumbnail{
		data:      thumbnailData,
		mediaType: mediaType,
	}

	videoThumbnails[videoMetaData.ID] = tnail
	thumbnailURL := "http://localhost:8091/api/thumbnails/" + videoMetaData.ID.String()
	fmt.Printf("Setting thumbnail URL to: %s\n", thumbnailURL)

	err = cfg.db.UpdateVideo(database.Video{
		ID:           videoMetaData.ID,
		ThumbnailURL: &thumbnailURL,
		CreateVideoParams: database.CreateVideoParams{
			Title:       videoMetaData.Title,
			Description: videoMetaData.Description,
			UserID:      videoMetaData.UserID,
		},
	})

	// Add this debug print

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating video", err)
		return
	}

	updatedVideo, err := cfg.db.GetVideo(videoID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error getting updated video", err)
		return
	}
	fmt.Printf("Updated video: %+v\n", updatedVideo)

	respondWithJSON(w, http.StatusOK, updatedVideo)
}
