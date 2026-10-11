package repository

import (
	"context"
	"errors"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Обработчики сравнивают с ней ошибку и отдают 404, не зная про GORM.
var ErrNotFound = gorm.ErrRecordNotFound

// Тип файла не подходит: обработчик отдаёт по ней 400.
var ErrUnsupportedMedia = errors.New("неподдерживаемый тип файла")

// Черновик у пользователя уже есть: обработчик отдаёт по ней 409.
var ErrDraftExists = errors.New("черновик уже есть")

type Settings struct {
	PostgresDSN string

	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioUseSSL    bool
	MinioPublicURL string
}

type Repository struct {
	db *gorm.DB

	minio          *minio.Client
	minioBucket    string
	minioPublicURL string
}

func New(s *Settings) (*Repository, error) {
	db, err := gorm.Open(postgres.Open(s.PostgresDSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	client, err := minio.New(s.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s.MinioAccessKey, s.MinioSecretKey, ""),
		Secure: s.MinioUseSSL,
	})
	if err != nil {
		return nil, err
	}

	// Бакет наполняется контейнером minio-init, но приложение не должно падать, если его нет.
	exists, err := client.BucketExists(context.Background(), s.MinioBucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err = client.MakeBucket(context.Background(), s.MinioBucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}

	return &Repository{
		db:             db,
		minio:          client,
		minioBucket:    s.MinioBucket,
		minioPublicURL: s.MinioPublicURL,
	}, nil
}
