package handler

import (
	"errors"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"iss-visibility/internal/app/auth"
	"iss-visibility/internal/app/dto"
	"iss-visibility/internal/app/repository"
)

const (
	minLatitudeLimit = -90.0
	maxLatitudeLimit = 90.0

	minLongitudeLimit = -180.0
	maxLongitudeLimit = 180.0

	maxMultipartMemory = 8 << 20
)

func pathID(ctx *gin.Context) (uint, bool) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || id <= 0 {
		ctx.Status(http.StatusBadRequest)
		return 0, false
	}

	return uint(id), true
}

// Единый ответ на запросы одной услуги: 404 на ненайденную, 500 на ошибку базы.
func (h *Handler) respondObservationPoint(
	ctx *gin.Context,
	point repository.ObservationPointWithLikes,
	err error,
) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		ctx.Status(http.StatusNotFound)
	case err != nil:
		h.errorHandler(ctx, http.StatusInternalServerError, err)
	default:
		ctx.JSON(http.StatusOK, dto.NewObservationPoint(point, auth.CurrentUserSingleton().ID))
	}
}

// GetObservationPoints GET /api/observation_points — список опубликованных услуг
// с фильтрацией по широте на бэкенде.
func (h *Handler) GetObservationPoints(ctx *gin.Context) {
	user := auth.CurrentUserSingleton()

	minLatitude := minLatitudeLimit
	if query := strings.TrimSpace(ctx.Query("min_latitude")); query != "" {
		value, err := strconv.ParseFloat(query, 64)
		if err != nil || value < minLatitudeLimit || value > maxLatitudeLimit {
			ctx.Status(http.StatusBadRequest)
			return
		}
		minLatitude = value
	}

	points, err := h.Repository.PublishedObservationPoints(user.ID, minLatitude)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.NewObservationPoints(points, user.ID))
}

// GetFeed GET /api/observation_points/feed — лента без идентификатора:
// база отдаёт ровно одну строку, первую опубликованную.
func (h *Handler) GetFeed(ctx *gin.Context) {
	point, err := h.Repository.FirstPublishedObservationPoint(auth.CurrentUserSingleton().ID)
	h.respondObservationPoint(ctx, point, err)
}

// GetObservationPoint GET /api/observation_points/:id — услуга по идентификатору,
// с ?next=true — следующая опубликованная, по кругу через пропуски в идентификаторах.
func (h *Handler) GetObservationPoint(ctx *gin.Context) {
	user := auth.CurrentUserSingleton()

	id, ok := pathID(ctx)
	if !ok {
		return
	}

	var (
		point repository.ObservationPointWithLikes
		err   error
	)

	if ctx.Query("next") == "true" {
		point, err = h.Repository.NextPublishedObservationPoint(user.ID, id)
	} else {
		point, err = h.Repository.PublishedObservationPointByID(user.ID, id)
	}

	h.respondObservationPoint(ctx, point, err)
}

// GetObservationPointDraft GET /api/observation_points/draft — черновик текущего
// пользователя, идентификатор не указывается.
func (h *Handler) GetObservationPointDraft(ctx *gin.Context) {
	point, err := h.Repository.DraftObservationPoint(auth.CurrentUserSingleton().ID)
	h.respondObservationPoint(ctx, point, err)
}

// Отсутствие файла не ошибка, ошибка разбора формы — ошибка.
func formFile(ctx *gin.Context, field string) (*multipart.FileHeader, error) {
	header, err := ctx.FormFile(field)
	if errors.Is(err, http.ErrMissingFile) {
		return nil, nil
	}

	return header, err
}

func (h *Handler) mediaError(ctx *gin.Context, err error) {
	if errors.Is(err, repository.ErrUnsupportedMedia) {
		ctx.Status(http.StatusBadRequest)
		return
	}

	h.errorHandler(ctx, http.StatusInternalServerError, err)
}

// CreateObservationPointDraft POST /api/observation_points — создание черновика
// вместе с файлами изображения и короткого видео. Файлы приходят в multipart-форме,
// ложатся в Minio, а в базу попадают только ссылки на них.
func (h *Handler) CreateObservationPointDraft(ctx *gin.Context) {
	user := auth.CurrentUserSingleton()

	if err := ctx.Request.ParseMultipartForm(maxMultipartMemory); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(ctx.Request.FormValue("name"))
	description := strings.TrimSpace(ctx.Request.FormValue("description"))
	if name == "" || description == "" {
		ctx.Status(http.StatusBadRequest)
		return
	}

	image, err := formFile(ctx, "image")
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	video, err := formFile(ctx, "video")
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	// Тип файлов проверяется до создания записи: отказ не должен оставить пустой черновик.
	if image != nil {
		if err = repository.ValidateImage(image); err != nil {
			h.mediaError(ctx, err)
			return
		}
	}

	if video != nil {
		if err = repository.ValidateVideo(video); err != nil {
			h.mediaError(ctx, err)
			return
		}
	}

	point, err := h.Repository.CreateObservationPointDraft(user.ID, name, description)
	switch {
	case errors.Is(err, repository.ErrDraftExists):
		ctx.Status(http.StatusConflict)
		return
	case err != nil:
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	if image != nil {
		if err = h.Repository.SaveObservationPointImage(point.ID, image); err != nil {
			h.mediaError(ctx, err)
			return
		}
	}

	if video != nil {
		if err = h.Repository.SaveObservationPointVideo(point.ID, video); err != nil {
			h.mediaError(ctx, err)
			return
		}
	}

	// Перечитываем запись, чтобы в ответ попали ссылки на загруженные файлы.
	created, err := h.Repository.DraftObservationPoint(user.ID)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusCreated, dto.NewObservationPoint(created, user.ID))
}

type publishRequest struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

// PublishObservationPoint PUT /api/observation_points/:id/publish — смена статуса
// черновика на «опубликован» вместе с полями предметной области. Сам статус
// с клиента не принимается: его подставляет бизнес-логика.
func (h *Handler) PublishObservationPoint(ctx *gin.Context) {
	user := auth.CurrentUserSingleton()

	id, ok := pathID(ctx)
	if !ok {
		return
	}

	var request publishRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	if request.Latitude == nil || request.Longitude == nil ||
		*request.Latitude < minLatitudeLimit || *request.Latitude > maxLatitudeLimit ||
		*request.Longitude < minLongitudeLimit || *request.Longitude > maxLongitudeLimit {
		ctx.Status(http.StatusBadRequest)
		return
	}

	err := h.Repository.PublishObservationPoint(id, user.ID, *request.Latitude, *request.Longitude)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		ctx.Status(http.StatusNotFound)
		return
	case err != nil:
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	point, err := h.Repository.PublishedObservationPointByID(user.ID, id)
	h.respondObservationPoint(ctx, point, err)
}

// DeleteObservationPoint DELETE /api/observation_points/:id — только логическое
// удаление и только своей услуги.
func (h *Handler) DeleteObservationPoint(ctx *gin.Context) {
	user := auth.CurrentUserSingleton()

	id, ok := pathID(ctx)
	if !ok {
		return
	}

	err := h.Repository.DeleteObservationPoint(id, user.ID)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		ctx.Status(http.StatusNotFound)
	case err != nil:
		h.errorHandler(ctx, http.StatusInternalServerError, err)
	default:
		ctx.Status(http.StatusNoContent)
	}
}

type likeRequest struct {
	Value *int `json:"value"`
}

// SetObservationPointLike POST /api/observation_points/:id/like — отметка текущего
// пользователя: 1 ставит, 0 снимает.
func (h *Handler) SetObservationPointLike(ctx *gin.Context) {
	user := auth.CurrentUserSingleton()

	id, ok := pathID(ctx)
	if !ok {
		return
	}

	var request likeRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	if request.Value == nil || (*request.Value != 0 && *request.Value != 1) {
		ctx.Status(http.StatusBadRequest)
		return
	}

	err := h.Repository.SetObservationPointLike(user.ID, id, *request.Value)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		ctx.Status(http.StatusNotFound)
		return
	case err != nil:
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	point, err := h.Repository.PublishedObservationPointByID(user.ID, id)
	h.respondObservationPoint(ctx, point, err)
}
