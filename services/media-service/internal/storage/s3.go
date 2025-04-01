package storage

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"services/media-service/internal/config"
	"services/media-service/internal/model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/google/uuid"
	"golang.org/x/image/draw"
)

// S3Storage implements the Storage interface for Amazon S3
type S3Storage struct {
	bucket         string
	baseURL        string
	s3Client       *s3.S3
	s3Uploader     *s3manager.Uploader
	s3Downloader   *s3manager.Downloader
	thumbnailSizes []config.ThumbnailSize
}

// NewS3Storage creates a new S3Storage
func NewS3Storage(cfg *config.S3StorageConfig, thumbnailSizes []config.ThumbnailSize) (*S3Storage, error) {
	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(cfg.Region),
		Credentials: credentials.NewStaticCredentials(cfg.AccessKey, cfg.SecretKey, ""),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	// Create S3 client
	s3Client := s3.New(sess)

	// Create a bucket if it doesn't exist
	_, err = s3Client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(cfg.Bucket),
	})

	if err != nil {
		// If the bucket doesn't exist, create it
		_, err = s3Client.CreateBucket(&s3.CreateBucketInput{
			Bucket: aws.String(cfg.Bucket),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create S3 bucket: %w", err)
		}
	}

	// Create base URL if not provided
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.Bucket, cfg.Region)
	}

	return &S3Storage{
		bucket:         cfg.Bucket,
		baseURL:        baseURL,
		s3Client:       s3Client,
		s3Uploader:     s3manager.NewUploader(sess),
		s3Downloader:   s3manager.NewDownloader(sess),
		thumbnailSizes: thumbnailSizes,
	}, nil
}

// Store saves a file to S3
func (s *S3Storage) Store(ctx context.Context, file *multipart.FileHeader, purpose string, entityID string) (*model.MediaFile, error) {
	// Generate unique ID
	id := uuid.New().String()

	// Extract file extension
	ext := filepath.Ext(file.Filename)
	if ext == "" {
		ext = ".bin" // Default extension if none provided
	}

	// Generate key (path in S3)
	filename := fmt.Sprintf("%s%s", id, ext)
	key := fmt.Sprintf("%s/%s/%s", purpose, entityID, filename)

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Determine content type
	contentType := file.Header.Get("Content-Type")

	// Determine image dimensions if it's an image
	var width, height int
	var imgData []byte

	if isImage(contentType) {
		// Read the entire file into memory
		imgData, err = io.ReadAll(src)
		if err != nil {
			return nil, fmt.Errorf("failed to read image data: %w", err)
		}

		// Decode the image to get dimensions
		img, _, err := image.Decode(bytes.NewReader(imgData))
		if err == nil {
			bounds := img.Bounds()
			width = bounds.Dx()
			height = bounds.Dy()
		}

		// Reset the src reader position (if possible)
		if seeker, ok := src.(io.Seeker); ok {
			_, err = seeker.Seek(0, 0)
			if err != nil {
				// If we can't seek back to the beginning, use the in-memory data
				_, err = s.s3Uploader.Upload(&s3manager.UploadInput{
					Bucket:      aws.String(s.bucket),
					Key:         aws.String(key),
					Body:        bytes.NewReader(imgData),
					ContentType: aws.String(contentType),
					ACL:         aws.String("public-read"),
				})
				if err != nil {
					return nil, fmt.Errorf("failed to upload file to S3: %w", err)
				}

				// Skip the regular upload since we've already done it
				goto UploadComplete
			}
		} else {
			// If src doesn't support seeking, use the in-memory data
			_, err = s.s3Uploader.Upload(&s3manager.UploadInput{
				Bucket:      aws.String(s.bucket),
				Key:         aws.String(key),
				Body:        bytes.NewReader(imgData),
				ContentType: aws.String(contentType),
				ACL:         aws.String("public-read"),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to upload file to S3: %w", err)
			}

			// Skip the regular upload since we've already done it
			goto UploadComplete
		}
	}

	// Upload to S3 using the original src
	_, err = s.s3Uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        src,
		ContentType: aws.String(contentType),
		ACL:         aws.String("public-read"),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to upload file to S3: %w", err)
	}

UploadComplete:
	// Create media file metadata
	media := &model.MediaFile{
		ID:          id,
		FileName:    file.Filename,
		ContentType: contentType,
		Size:        file.Size,
		URL:         fmt.Sprintf("%s/%s", s.baseURL, key),
		Width:       width,
		Height:      height,
		CreatedAt:   time.Now(),
		StorageType: "s3",
		StoragePath: key,
	}

	return media, nil
}

// Get retrieves a file from S3
func (s *S3Storage) Get(ctx context.Context, id string) (io.ReadCloser, *model.MediaFile, error) {
	// We need to list objects with the prefix to find the full key
	resp, err := s.s3Client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(id),
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to list objects in S3: %w", err)
	}

	if len(resp.Contents) == 0 {
		return nil, nil, fmt.Errorf("file not found: %s", id)
	}

	// Get the first matching object (there should only be one with this prefix)
	obj := resp.Contents[0]
	key := *obj.Key

	// Get the object
	getResp, err := s.s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to get object from S3: %w", err)
	}

	// Extract file info
	contentType := "application/octet-stream"
	if getResp.ContentType != nil {
		contentType = *getResp.ContentType
	}

	size := int64(0)
	if getResp.ContentLength != nil {
		size = *getResp.ContentLength
	}

	// Extract filename from key
	filename := filepath.Base(key)

	// Create media file metadata
	media := &model.MediaFile{
		ID:          id,
		FileName:    filename,
		ContentType: contentType,
		Size:        size,
		URL:         fmt.Sprintf("%s/%s", s.baseURL, key),
		CreatedAt:   *getResp.LastModified,
		StorageType: "s3",
		StoragePath: key,
	}

	return getResp.Body, media, nil
}

// Delete removes a file from S3
func (s *S3Storage) Delete(ctx context.Context, id string) error {
	// Find objects with this prefix
	resp, err := s.s3Client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(id),
	})

	if err != nil {
		return fmt.Errorf("failed to list objects in S3: %w", err)
	}

	if len(resp.Contents) == 0 {
		return fmt.Errorf("file not found: %s", id)
	}

	// Create delete objects input
	objects := make([]*s3.ObjectIdentifier, len(resp.Contents))
	for i, obj := range resp.Contents {
		objects[i] = &s3.ObjectIdentifier{
			Key: obj.Key,
		}
	}

	// Delete the objects
	_, err = s.s3Client.DeleteObjects(&s3.DeleteObjectsInput{
		Bucket: aws.String(s.bucket),
		Delete: &s3.Delete{
			Objects: objects,
			Quiet:   aws.Bool(false),
		},
	})

	if err != nil {
		return fmt.Errorf("failed to delete objects from S3: %w", err)
	}

	return nil
}

// GenerateThumbnails creates thumbnails for an image and returns their metadata
func (s *S3Storage) GenerateThumbnails(ctx context.Context, originalFile *model.MediaFile) ([]model.Thumbnail, error) {
	// Check if the file is an image
	if !isImage(originalFile.ContentType) {
		return nil, nil // Non-image files don't get thumbnails
	}

	// Get the original file
	getResp, err := s.s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(originalFile.StoragePath),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get original image from S3: %w", err)
	}
	defer getResp.Body.Close()

	// Read the entire image
	imgData, err := io.ReadAll(getResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	// Decode the image
	img, format, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	thumbnails := make([]model.Thumbnail, 0, len(s.thumbnailSizes))

	// Extract path components
	pathParts := strings.Split(originalFile.StoragePath, "/")
	fileName := pathParts[len(pathParts)-1]
	nameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	basePath := strings.Join(pathParts[:len(pathParts)-1], "/")

	// Generate each thumbnail
	for _, size := range s.thumbnailSizes {
		// Calculate dimensions while preserving aspect ratio
		width, height := calculateDimensions(originalFile.Width, originalFile.Height, size.Width, size.Height)

		// Create the thumbnail image
		thumbImg := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.BiLinear.Scale(thumbImg, thumbImg.Bounds(), img, img.Bounds(), draw.Over, nil)

		// Encode the thumbnail to a buffer
		var buf bytes.Buffer
		switch format {
		case "jpeg":
			if err := jpeg.Encode(&buf, thumbImg, &jpeg.Options{Quality: 85}); err != nil {
				return nil, fmt.Errorf("failed to encode JPEG thumbnail: %w", err)
			}
		case "png":
			if err := png.Encode(&buf, thumbImg); err != nil {
				return nil, fmt.Errorf("failed to encode PNG thumbnail: %w", err)
			}
		default:
			// Default to JPEG for other formats
			if err := jpeg.Encode(&buf, thumbImg, &jpeg.Options{Quality: 85}); err != nil {
				return nil, fmt.Errorf("failed to encode thumbnail: %w", err)
			}
		}

		// Generate key for thumbnail
		ext := filepath.Ext(fileName)
		thumbFileName := fmt.Sprintf("%s_%s%s", nameWithoutExt, size.Name, ext)
		thumbKey := fmt.Sprintf("%s/%s", basePath, thumbFileName)

		// Upload thumbnail to S3
		_, err = s.s3Uploader.Upload(&s3manager.UploadInput{
			Bucket:      aws.String(s.bucket),
			Key:         aws.String(thumbKey),
			Body:        bytes.NewReader(buf.Bytes()),
			ContentType: aws.String(originalFile.ContentType),
			ACL:         aws.String("public-read"),
		})

		if err != nil {
			return nil, fmt.Errorf("failed to upload thumbnail to S3: %w", err)
		}

		// Add thumbnail to list
		thumbnails = append(thumbnails, model.Thumbnail{
			Name:   size.Name,
			URL:    fmt.Sprintf("%s/%s", s.baseURL, thumbKey),
			Width:  width,
			Height: height,
		})
	}

	return thumbnails, nil
}
