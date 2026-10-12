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

// pathID разбирает идентификатор из адреса. Нечисловое значение и ноль — ошибка
// запроса, поэтому проверка сразу отвечает 400, а обработчику остаётся выйти.
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
		ctx.JSON(http.StatusOK, dto.NewObservationPoint(point))
	}
}

// То же для ленты: отличается только сериализатором, в её ответе есть признак liked.
func (h *Handler) respondFeedPoint(
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
		ctx.JSON(http.StatusOK, dto.NewObservationPointFeed(point))
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

	ctx.JSON(http.StatusOK, dto.NewObservationPointLists(points, user.ID))
}

// GetFeed GET /api/observation_points/feed — лента, всегда ровно одна запись.
// Без параметра отдаётся первая опубликованная услуга, с ?after=<id> — следующая
// за указанной; когда следующей нет, лента замыкается на первую. Идентификатор
// приходит параметром запроса, а не частью адреса: отдельного метода «услуга по
// идентификатору» в сервисе нет.
func (h *Handler) GetFeed(ctx *gin.Context) {
	user := auth.CurrentUserSingleton()

	var (
		point repository.ObservationPointWithLikes
		err   error
	)

	if afterStr := strings.TrimSpace(ctx.Query("after")); afterStr == "" {
		point, err = h.Repository.FirstPublishedObservationPoint(user.ID)
	} else {
		var afterID uint64
		afterID, err = strconv.ParseUint(afterStr, 10, 64)
		if err != nil || afterID == 0 {
			ctx.Status(http.StatusBadRequest)
			return
		}

		point, err = h.Repository.NextPublishedObservationPoint(user.ID, uint(afterID))
	}

	h.respondFeedPoint(ctx, point, err)
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

// CreateObservationPointDraft POST /api/observation_points — создание черновика:
// название и файлы изображения и короткого видео. Файлы приходят в multipart-форме,
// ложатся в Minio, а в базу попадают только имена, сгенерированные на латинице.
// Описание здесь не принимается — оно задаётся при публикации.
func (h *Handler) CreateObservationPointDraft(ctx *gin.Context) {
	user := auth.CurrentUserSingleton()

	if err := ctx.Request.ParseMultipartForm(maxMultipartMemory); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(ctx.Request.FormValue("name"))
	if name == "" {
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

	point, err := h.Repository.CreateObservationPointDraft(user.ID, name)
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

	ctx.JSON(http.StatusCreated, dto.NewObservationPoint(created))
}

// publishRequest — тело публикации. Координаты описаны указателями, чтобы отличить
// пропущенное поле от переданного нуля: нулевая широта и долгота допустимы.
type publishRequest struct {
	Description string   `json:"description"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
}

// PublishObservationPoint PUT /api/observation_points/:id/publish — смена статуса
// черновика на «опубликован» вместе с описанием и полями предметной области. Сам
// статус с клиента не принимается: его подставляет бизнес-логика. Публикация
// необратима — метода возврата в черновик в сервисе нет.
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

	description := strings.TrimSpace(request.Description)
	if description == "" {
		ctx.Status(http.StatusBadRequest)
		return
	}

	if request.Latitude == nil || request.Longitude == nil ||
		*request.Latitude < minLatitudeLimit || *request.Latitude > maxLatitudeLimit ||
		*request.Longitude < minLongitudeLimit || *request.Longitude > maxLongitudeLimit {
		ctx.Status(http.StatusBadRequest)
		return
	}

	err := h.Repository.PublishObservationPoint(id, user.ID, description, *request.Latitude, *request.Longitude)
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
