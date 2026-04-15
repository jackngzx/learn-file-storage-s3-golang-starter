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
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to find the video in the database", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "user is not the author of the video", err)
		return
	}

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid content type", err)
		return
	}
	if mediaType != "image/png" && mediaType != "image/jpeg" {
		respondWithError(w, http.StatusBadRequest, "invalid mediaType, only png or jpeg allowed", err)
		return
	}
	parts := strings.Split(mediaType, "/")
	ext := "." + parts[1]
	key := make([]byte, 32)
	rand.Read(key)
	fileNameString := base64.RawURLEncoding.EncodeToString(key) + ext
	fileImgPath := filepath.Join(cfg.assetsRoot, fileNameString)
	f, err := os.Create(fileImgPath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to store file on file system", err)
		return
	}
	defer f.Close()
	if _, err := io.Copy(f, file); err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to copy form file to local file", err)
		return
	}

	dataURL := fmt.Sprintf("http://localhost:%s/%s", cfg.port, fileImgPath)
	video.ThumbnailURL = &dataURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to update video thumbnail url", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
