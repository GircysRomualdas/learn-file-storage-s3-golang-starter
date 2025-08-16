package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
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

	// TODO: implement the upload here

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to get video", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusForbidden, "Not authorized to upload thumbnail", nil)
		return
	}
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse from file", err)
		return
	}
	defer file.Close()
	headercontentType := header.Header.Get("Content-Type")
	if headercontentType == "" {
		respondWithError(w, http.StatusBadRequest, "Missing Content-Type from thumbnail", nil)
		return
	}

	mediaType, _, err := mime.ParseMediaType(headercontentType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type", err)
		return
	}
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Only JPEG and PNG thumbnails are allowed", nil)
		return
	}

	extention := strings.TrimPrefix(mediaType, "image/")
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to generate random bytes", err)
		return
	}
	name := base64.RawURLEncoding.EncodeToString(b)
	fileName := fmt.Sprintf("%s.%s", name, extention)
	path := filepath.Join(cfg.assetsRoot, fileName)
	newFile, err := os.Create(path)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create file", err)
		return
	}
	defer newFile.Close()
	if _, err := io.Copy(newFile, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to copy file", err)
		return
	}
	urlPath := fmt.Sprintf("/assets/%s", fileName)
	dataURL := fmt.Sprintf("http://localhost:%s%s", cfg.port, urlPath)

	video.ThumbnailURL = &dataURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
