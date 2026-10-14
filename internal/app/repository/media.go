package repository

import (
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/minio/minio-go/v7"

	"iss-visibility/internal/app/ds"
)

// Тип файла определяется по содержимому, а не по расширению из запроса.
var allowedImages = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

var allowedVideos = map[string]string{
	"video/mp4":       ".mp4",
	"video/webm":      ".webm",
	"video/quicktime": ".mov",
}

// Имя объекта собирается на латинице из вида файла и идентификатора услуги.
func objectName(kind string, pointID uint, extension string) string {
	return fmt.Sprintf("observation_point_%d_%s%s", pointID, kind, extension)
}

func detectContentType(header *multipart.FileHeader) (string, error) {
	file, err := header.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	buffer := make([]byte, 512)
	read, err := file.Read(buffer)
	if err != nil && read == 0 {
		return "", err
	}

	return http.DetectContentType(buffer[:read]), nil
}

func mediaType(header *multipart.FileHeader, allowed map[string]string) (string, string, error) {
	contentType, err := detectContentType(header)
	if err != nil {
		return "", "", err
	}

	extension, ok := allowed[contentType]
	if !ok {
		return "", "", ErrUnsupportedMedia
	}

	return contentType, extension, nil
}

// ValidateImage проверяет тип файла до создания записи, чтобы отказ
// не оставил в базе пустой черновик. То же делает ValidateVideo.
func ValidateImage(header *multipart.FileHeader) error {
	_, _, err := mediaType(header, allowedImages)

	return err
}

func ValidateVideo(header *multipart.FileHeader) error {
	_, _, err := mediaType(header, allowedVideos)

	return err
}

// Файл кладётся в хранилище, в базу попадает только ссылка на него.
func (r *Repository) uploadMedia(
	kind string,
	pointID uint,
	header *multipart.FileHeader,
	allowed map[string]string,
) (string, error) {
	contentType, extension, err := mediaType(header, allowed)
	if err != nil {
		return "", err
	}

	file, err := header.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	name := objectName(kind, pointID, extension)

	_, err = r.minio.PutObject(
		context.Background(),
		r.minioBucket,
		name,
		file,
		header.Size,
		minio.PutObjectOptions{ContentType: contentType},
	)
	if err != nil {
		return "", err
	}

	return strings.TrimRight(r.minioPublicURL, "/") + "/" + name, nil
}

func (r *Repository) SaveObservationPointImage(pointID uint, header *multipart.FileHeader) error {
	url, err := r.uploadMedia("image", pointID, header, allowedImages)
	if err != nil {
		return err
	}

	return r.db.Model(&ds.ObservationPoint{}).
		Where("id = ?", pointID).
		UpdateColumn("image_url", url).Error
}

func (r *Repository) SaveObservationPointVideo(pointID uint, header *multipart.FileHeader) error {
	url, err := r.uploadMedia("video", pointID, header, allowedVideos)
	if err != nil {
		return err
	}

	return r.db.Model(&ds.ObservationPoint{}).
		Where("id = ?", pointID).
		UpdateColumn("video_url", url).Error
}
