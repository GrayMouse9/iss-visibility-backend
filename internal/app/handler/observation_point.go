package handler

import (
	"errors"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

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

// GetObservationPoints GET /api/observation_points — список опубликованных услуг.
//
// @Summary Список точек наблюдения
// @Description Опубликованные услуги с фильтром по минимальной широте. Токен необязателен: без него признак is_mine всегда 0.
// @Tags Точки наблюдения
// @Produce json
// @Security BearerAuth
// @Param min_latitude query number false "Минимальная широта, от -90 до 90"
// @Success 200 {array} dto.ObservationPointList
// @Failure 400 "Недопустимое значение широты"
// @Failure 401 "Предъявлен негодный токен"
// @Failure 500
// @Router /observation_points [get]
func (h *Handler) GetObservationPoints(ctx *gin.Context) {
	userID := currentUserID(ctx)

	minLatitude := minLatitudeLimit
	if query := strings.TrimSpace(ctx.Query("min_latitude")); query != "" {
		value, err := strconv.ParseFloat(query, 64)
		if err != nil || value < minLatitudeLimit || value > maxLatitudeLimit {
			ctx.Status(http.StatusBadRequest)
			return
		}
		minLatitude = value
	}

	points, err := h.Repository.PublishedObservationPoints(userID, minLatitude)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.NewObservationPointLists(points, userID))
}

// GetFeed GET /api/observation_points/feed — лента, всегда ровно одна запись.
//
// @Summary Лента
// @Description Ровно одна услуга: без параметра — первая, с after — следующая за указанной. Когда следующей нет, лента замыкается на первую.
// @Tags Точки наблюдения
// @Produce json
// @Security BearerAuth
// @Param after query integer false "Идентификатор предыдущей показанной услуги"
// @Success 200 {object} dto.ObservationPointFeed
// @Failure 400 "Недопустимый идентификатор"
// @Failure 401 "Предъявлен негодный токен"
// @Failure 404 "Опубликованных услуг нет"
// @Failure 500
// @Router /observation_points/feed [get]
func (h *Handler) GetFeed(ctx *gin.Context) {
	userID := currentUserID(ctx)

	var (
		point repository.ObservationPointWithLikes
		err   error
	)

	if afterStr := strings.TrimSpace(ctx.Query("after")); afterStr == "" {
		point, err = h.Repository.FirstPublishedObservationPoint(userID)
	} else {
		var afterID uint64
		afterID, err = strconv.ParseUint(afterStr, 10, 64)
		if err != nil || afterID == 0 {
			ctx.Status(http.StatusBadRequest)
			return
		}

		point, err = h.Repository.NextPublishedObservationPoint(userID, uint(afterID))
	}

	h.respondFeedPoint(ctx, point, err)
}

// GetObservationPointDraft GET /api/observation_points/draft — черновик пользователя.
//
// @Summary Черновик текущего пользователя
// @Description Идентификатор не указывается: черновик у пользователя может быть только один.
// @Tags Точки наблюдения
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.ObservationPoint
// @Failure 401
// @Failure 404 "Черновика нет"
// @Failure 500
// @Router /observation_points/draft [get]
func (h *Handler) GetObservationPointDraft(ctx *gin.Context) {
	point, err := h.Repository.DraftObservationPoint(currentUserID(ctx))
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

// CreateObservationPointDraft POST /api/observation_points — создание черновика.
//
// @Summary Создание черновика
// @Description Название и файлы. Файлы уходят в Minio, в базу попадают только ссылки. Описание задаётся при публикации.
// @Tags Точки наблюдения
// @Accept mpfd
// @Produce json
// @Security BearerAuth
// @Param name formData string true "Название точки наблюдения"
// @Param image formData file false "Изображение"
// @Param video formData file false "Короткое видео"
// @Success 201 {object} dto.ObservationPoint
// @Failure 400 "Пустое название или недопустимый тип файла"
// @Failure 401
// @Failure 409 "Черновик уже существует"
// @Failure 500
// @Router /observation_points [post]
func (h *Handler) CreateObservationPointDraft(ctx *gin.Context) {
	userID := currentUserID(ctx)

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

	point, err := h.Repository.CreateObservationPointDraft(userID, name)
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
	created, err := h.Repository.DraftObservationPoint(userID)
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

// PublishObservationPoint PUT /api/observation_points/:id/publish — публикация черновика.
//
// @Summary Публикация услуги
// @Description Переводит черновик в опубликованные вместе с описанием и координатами. Статус с клиента не принимается, операция необратима.
// @Tags Точки наблюдения
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path integer true "Идентификатор черновика"
// @Param request body handler.publishRequest true "Описание и координаты"
// @Success 200 {object} dto.ObservationPoint
// @Failure 400 "Пустое описание или координаты вне допустимых границ"
// @Failure 401
// @Failure 404 "Черновик не найден или принадлежит другому пользователю"
// @Failure 500
// @Router /observation_points/{id}/publish [put]
func (h *Handler) PublishObservationPoint(ctx *gin.Context) {
	userID := currentUserID(ctx)

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

	err := h.Repository.PublishObservationPoint(id, userID, description, *request.Latitude, *request.Longitude)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		ctx.Status(http.StatusNotFound)
		return
	case err != nil:
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	point, err := h.Repository.PublishedObservationPointByID(userID, id)
	h.respondObservationPoint(ctx, point, err)
}

// DeleteObservationPoint DELETE /api/observation_points/:id — логическое удаление.
//
// @Summary Удаление услуги
// @Description Логическое удаление: запись остаётся в базе со статусом «удалена».
// @Tags Точки наблюдения
// @Security BearerAuth
// @Param id path integer true "Идентификатор услуги"
// @Success 204 "Услуга удалена"
// @Failure 400 "Недопустимый идентификатор"
// @Failure 401
// @Failure 404 "Услуга не найдена или принадлежит другому пользователю"
// @Failure 500
// @Router /observation_points/{id} [delete]
func (h *Handler) DeleteObservationPoint(ctx *gin.Context) {
	userID := currentUserID(ctx)

	id, ok := pathID(ctx)
	if !ok {
		return
	}

	err := h.Repository.DeleteObservationPoint(id, userID)
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

// SetObservationPointLike POST /api/observation_points/:id/like — отметка «нравится».
//
// @Summary Отметка «нравится»
// @Description Значение 1 ставит отметку текущего пользователя, 0 снимает. Повторный вызов с тем же значением ничего не меняет.
// @Tags Точки наблюдения
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path integer true "Идентификатор услуги"
// @Param request body handler.likeRequest true "Значение отметки: 0 или 1"
// @Success 200 {object} dto.ObservationPoint
// @Failure 400 "Значение отличается от 0 и 1"
// @Failure 401
// @Failure 404 "Услуга не найдена"
// @Failure 500
// @Router /observation_points/{id}/like [post]
func (h *Handler) SetObservationPointLike(ctx *gin.Context) {
	userID := currentUserID(ctx)

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

	err := h.Repository.SetObservationPointLike(userID, id, *request.Value)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		ctx.Status(http.StatusNotFound)
		return
	case err != nil:
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	point, err := h.Repository.PublishedObservationPointByID(userID, id)
	h.respondObservationPoint(ctx, point, err)
}
