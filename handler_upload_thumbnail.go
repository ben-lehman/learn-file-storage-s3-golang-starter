package main
 
import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

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

  const maxMemory = 10 << 20
  r.ParseMultipartForm(maxMemory)
  
  file, header, err := r.FormFile("thumbnail")
  if err != nil {
    respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
    return
  }
  defer file.Close()
  mediaType := header.Header.Values("Content-Type")[0]

  video, err := cfg.db.GetVideo(videoID)
  if err != nil {
    respondWithError(w, http.StatusNotFound, "Could not get video from db", err)
    return
  }
  if userID != video.UserID {
    respondWithError(w, http.StatusUnauthorized, "Unauthroized", err)
    return
  }
 
  tnFileType, _, err := mime.ParseMediaType(mediaType)
  if err != nil {
    respondWithError(w, http.StatusBadRequest, "Invalid image Content-Type", err)
    return
  }
  if tnFileType != "image/jpeg" && tnFileType != "image/png" {
    respondWithError(w, http.StatusBadRequest, "Invalid image Content-Type", nil)
    return
  }

  b32 := make([]byte, 32)
  rand.Read(b32)
  randName := base64.RawURLEncoding.EncodeToString(b32)
  
  assetPath := getAssetPath(randName, tnFileType)
  assetDiskPath := cfg.getAssetDiskPath(assetPath)
  tnFile, err := os.Create(assetDiskPath)
  if err != nil {
    respondWithError(w, http.StatusInternalServerError, "Error creating thumbnail file", err)
   return 
  }
  defer tnFile.Close()

  _, err = io.Copy(tnFile, file)
  if err != nil {
    respondWithError(w, http.StatusInternalServerError, "Unable to write file", err)
    return
  }

  url := cfg.getAssetURL(assetPath)
  video.ThumbnailURL = &url

  err = cfg.db.UpdateVideo(video)
  if err != nil {
    respondWithError(w, http.StatusInternalServerError, "Unable to create video", err)
    return
  }

	respondWithJSON(w, http.StatusOK, video)
}
