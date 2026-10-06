package main

import (
	"log"
	"os"

	"github.com/cloudinary/cloudinary-go/v2"
)

func initCloudinary() *cloudinary.Cloudinary {
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		log.Fatal("Cloudinary environment variables are not set")
	}

	cld, err := cloudinary.NewFromParams(
		cloudName,
		apiKey,
		apiSecret,
	)
	if err != nil {
		log.Fatal("Unable to initialize Cloudinary:", err)
	}

	return cld
}
