package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
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
  
	fmt.Println("uploading video", videoID, "by user", userID)

  video, err := cfg.db.GetVideo(videoID)
  if err != nil {
    respondWithError(w, http.StatusNotFound, "Could not get video from db", err)
    return
  }
  if userID != video.UserID {
    respondWithError(w, http.StatusUnauthorized, "Unauthroized", err)
    return
  }

  const maxMemory = 1 << 30
  r.ParseMultipartForm(maxMemory)
  
  file, header, err := r.FormFile("video")
  if err != nil {
    respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
    return
  }
  defer file.Close()

  mediaType := header.Header.Values("Content-Type")[0]
  fileType, _, err := mime.ParseMediaType(mediaType)
  if err != nil {
    respondWithError(w, http.StatusBadRequest, "Invalid image Content-Type", err)
    return
  }
  if fileType != "video/mp4" {
    respondWithError(w, http.StatusBadRequest, "Invalid image Content-Type", nil)
    return
  }

  tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
  if err != nil {
    respondWithError(w, http.StatusInternalServerError, "Error creating video file", err)
   return 
  }
  defer os.Remove(tempFile.Name())
  defer tempFile.Close()

  _, err = io.Copy(tempFile, file)
  if err != nil {
    respondWithError(w, http.StatusInternalServerError, "Unable to write file", err)
    return
  }
  tempFile.Seek(0, io.SeekStart)

  
  aspectRatio, err := getVideoAspectRatio(tempFile.Name())
  if err != nil {
    respondWithError(w, http.StatusInternalServerError, "Unable to get aspect ratio", err)
    return
  }
  processedVideoFilePath, err := processVideoForFastStart(tempFile.Name())
  if err != nil {
    respondWithError(w, http.StatusInternalServerError, "Error processing video", err)
    return
  }
  processVideoFile, err := os.Open(processedVideoFilePath)
  if err != nil {
    respondWithError(w, http.StatusInternalServerError, "Unable to open processed video file", err)
    return
  }
  defer processVideoFile.Close()
  
  b32 := make([]byte, 32)
  rand.Read(b32)
  randName := base64.RawURLEncoding.EncodeToString(b32)
  randKey := aspectRatio + "/" + randName + mediaTypeToExt(fileType) 

  _, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
    Bucket: &cfg.s3Bucket,
    Key: &randKey,
    Body: processVideoFile,
    ContentType: &fileType,
  })
  if err != nil {
    respondWithError(w, http.StatusInternalServerError, "Unable to put object in s3 storage", err)
    return
  }

  videoURL := fmt.Sprintf("%s,%s", cfg.s3Bucket, randKey)  
  video.VideoURL = &videoURL;

  video, err = cfg.dbVideoToSignedVideo(video)
  if err != nil {
    respondWithError(w, http.StatusInternalServerError, "Unable to get signed URL", err)
    return
  }

  err = cfg.db.UpdateVideo(video)
  if err != nil {
    respondWithError(w, http.StatusInternalServerError, "Unable to create video", err)
    return
  }

	respondWithJSON(w, http.StatusOK, video)
}

