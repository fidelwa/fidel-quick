package storage

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// presignedURLTTL es la vida útil de las URLs firmadas que se devuelven al
// cliente para acceder a fotos de facturas. 1h es suficiente para que el
// admin abra el link en el panel y poco para limitar el blast-radius si se
// filtra.
const presignedURLTTL = time.Hour

type S3Client struct {
	client *minio.Client
	bucket string
}

func NewS3Client(endpoint, accessKey, secretKey, bucket, region string, useSSL bool) (*S3Client, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
		Region: region,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &S3Client{client: client, bucket: bucket}, nil
}

// Upload sube un archivo y devuelve una URL firmada con TTL = presignedURLTTL.
// La URL incluye la signature; solo es válida durante ese período.
func (s *S3Client) Upload(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	reader := bytes.NewReader(data)
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("upload to s3: %w", err)
	}

	signed, err := s.client.PresignedGetObject(ctx, s.bucket, key, presignedURLTTL, url.Values{})
	if err != nil {
		return "", fmt.Errorf("presign s3 url: %w", err)
	}
	return signed.String(), nil
}

// PresignKey devuelve una URL firmada para una key ya existente. Útil para
// re-firmar una URL cuando expira o para listar invoices viejas.
func (s *S3Client) PresignKey(ctx context.Context, key string) (string, error) {
	signed, err := s.client.PresignedGetObject(ctx, s.bucket, key, presignedURLTTL, url.Values{})
	if err != nil {
		return "", fmt.Errorf("presign s3 url: %w", err)
	}
	return signed.String(), nil
}
