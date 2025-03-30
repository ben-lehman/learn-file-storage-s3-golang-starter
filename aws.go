package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func generatePresignedURL (s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
  presignClient := s3.NewPresignClient(s3Client)

  req, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
    Bucket: &bucket,
    Key: &key,
  }, s3.WithPresignExpires(expireTime))
  if err != nil {
    return "", err
  }
  return req.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
  if video.VideoURL == nil {
    return video, nil
  }
  log.Println("Video URL from presigning...", video.VideoURL, *video.VideoURL)
  videoSplit := strings.Split(*video.VideoURL, ",")
  if len(videoSplit) != 2 {
    return video, nil
  }

  bucket := videoSplit[0]
  key := videoSplit[1]
  
  signedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, 10 * time.Minute)
  if err != nil {
    return database.Video{}, err
  }

  video.VideoURL = &signedURL

  return video, nil
}
