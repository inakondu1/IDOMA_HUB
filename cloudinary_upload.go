package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/cloudinary/cloudinary-go/v2/api"
	"github.com/cloudinary/cloudinary-go/v2/api/admin"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/cloudinary/cloudinary-go/v2/config"
)

func uploadToCloudinary(reader io.Reader, folder string, resourceType string) (string, error) {
	result, err := cld.Upload.Upload(
		context.Background(),
		reader,
		uploader.UploadParams{
			Folder:       folder,
			ResourceType: resourceType,
		},
	)

	if err != nil {
		return "", err
	}

	return result.SecureURL, nil
}

// uploadPostMediaToCloudinary uploads post media and returns its URL and public ID.
func uploadPostMediaToCloudinary(reader io.Reader, folder string, resourceType string) (string, string, error) {
	result, err := cld.Upload.Upload(
		context.Background(),
		reader,
		uploader.UploadParams{
			Folder:       folder,
			ResourceType: resourceType,
		},
	)
	if err != nil {
		return "", "", err
	}

	return result.SecureURL, result.PublicID, nil
}

func deleteStoryVideo(publicID string) {
	_, err := cld.Upload.Destroy(
		context.Background(),
		uploader.DestroyParams{
			PublicID:     publicID,
			Type:         "upload",
			ResourceType: "video",
			Invalidate:   api.Bool(true),
		},
	)
	if err != nil {
		log.Printf("Unable to delete rejected story video %s: %v", publicID, err)
	}
}

// deletePostMediaFromCloudinary removes a post image or video by its stored public ID.
func deletePostMediaFromCloudinary(publicID string, resourceType string) {
	if publicID == "" {
		return
	}

	if resourceType != "image" && resourceType != "video" {
		log.Printf("Skipping Cloudinary deletion for unsupported post media type %q (public ID %s)", resourceType, publicID)
		return
	}

	_, err := cld.Upload.Destroy(
		context.Background(),
		uploader.DestroyParams{
			PublicID:     publicID,
			Type:         "upload",
			ResourceType: resourceType,
			Invalidate:   api.Bool(true),
		},
	)
	if err != nil {
		log.Printf("Unable to delete post media %s: %v", publicID, err)
	}
}

func uploadStoryVideo(reader io.Reader) (string, float64, error) {
	ctx := context.Background()

	result, err := cld.Upload.Upload(
		ctx,
		reader,
		uploader.UploadParams{
			Folder:        "idoma_hub/statuses",
			ResourceType:  "video",
			MediaMetadata: api.Bool(true),
		},
	)
	if err != nil {
		return "", 0, err
	}

	conf, err := config.NewFromParams(
		os.Getenv("CLOUDINARY_CLOUD_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
	)
	if err != nil {
		deleteStoryVideo(result.PublicID)
		return "", 0, err
	}

	adminAPI, err := admin.NewWithConfiguration(conf)
	if err != nil {
		deleteStoryVideo(result.PublicID)
		return "", 0, err
	}

	asset, err := adminAPI.Asset(
		ctx,
		admin.AssetParams{
			AssetType:     api.Video,
			DeliveryType:  api.Upload,
			PublicID:      result.PublicID,
			MediaMetadata: api.Bool(true),
		},
	)
	if err != nil {
		_, deleteErr := cld.Upload.Destroy(
			ctx,
			uploader.DestroyParams{
				PublicID:     result.PublicID,
				Type:         "upload",
				ResourceType: "video",
				Invalidate:   api.Bool(true),
			},
		)
		if deleteErr != nil {
			log.Printf("Unable to delete story video after metadata lookup failed: %v", deleteErr)
		}
		return "", 0, err
	}

	rawDuration, ok := asset.VideoMetadata["duration"]
	if !ok {
		_, deleteErr := cld.Upload.Destroy(
			ctx,
			uploader.DestroyParams{
				PublicID:     result.PublicID,
				Type:         "upload",
				ResourceType: "video",
				Invalidate:   api.Bool(true),
			},
		)
		if deleteErr != nil {
			log.Printf("Unable to delete story video with missing duration metadata: %v", deleteErr)
		}
		return "", 0, fmt.Errorf("video duration metadata is unavailable")
	}

	var duration float64
	switch value := rawDuration.(type) {
	case float64:
		duration = value
	case float32:
		duration = float64(value)
	case int:
		duration = float64(value)
	case int64:
		duration = float64(value)
	case json.Number:
		duration, err = value.Float64()
		if err != nil {
			deleteStoryVideo(result.PublicID)
			return "", 0, err
		}
	default:
		deleteStoryVideo(result.PublicID)
		return "", 0, fmt.Errorf("unexpected video duration metadata type %T", rawDuration)
	}

	if duration > 300 {
		_, deleteErr := cld.Upload.Destroy(
			ctx,
			uploader.DestroyParams{
				PublicID:     result.PublicID,
				Type:         "upload",
				ResourceType: "video",
				Invalidate:   api.Bool(true),
			},
		)
		if deleteErr != nil {
			log.Printf("Unable to delete story video longer than five minutes: %v", deleteErr)
		}
		return "", duration, fmt.Errorf("story videos cannot exceed five minutes")
	}

	return result.SecureURL, duration, nil
}
