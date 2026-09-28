package main

import (
	"context"
	"io"

	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
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
