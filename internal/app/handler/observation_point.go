package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"iss-visibility/internal/app/repository"
)

const (
	minLatitudeLimit = -90.0
	maxLatitudeLimit = 90.0

	minLongitudeLimit = -180.0
	maxLongitudeLimit = 180.0

	// Пользователь задан жёстко: авторизация появится в лабораторной работе №4.
	CurrentUserID = 1

	// Файлы по умолчанию — если ссылки в базе пустые.
	defaultImageURL = "/img/default.jpg"
	defaultVideoURL = "/video/default.mp4"
)

// ImageURL и VideoURL перекрывают поля вложенной структуры: в них подставлены файлы по умолчанию.
type ObservationPointView struct {
	repository.ObservationPointWithLikes
	ImageURL string
	VideoURL string

	LatAbs       float64
	LatHemi      string
	LatHemiShort string
	LonAbs       float64
	LonHemi      string
	LonHemiShort string
}

func (h *Handler) observationPointView(p repository.ObservationPointWithLikes) ObservationPointView {
	image := p.ImageURL
	if image == "" {
		image = defaultImageURL
	}
	video := p.VideoURL
	if video == "" {
		video = defaultVideoURL
	}

	lat, latHemi, latShort := p.Latitude, "северная", "с. ш."
	if lat < 0 {
		lat, latHemi, latShort = -lat, "южная", "ю. ш."
	}
	lon, lonHemi, lonShort := p.Longitude, "восточная", "в. д."
	if lon < 0 {
		lon, lonHemi, lonShort = -lon, "западная", "з. д."
	}

	return ObservationPointView{
		ObservationPointWithLikes: p,
		ImageURL:                  image,
		VideoURL:                  video,
		LatAbs:                    lat,
		LatHemi:                   latHemi,
		LatHemiShort:              latShort,
		LonAbs:                    lon,
		LonHemi:                   lonHemi,
		LonHemiShort:              lonShort,
	}
}

func (h *Handler) GetFeed(ctx *gin.Context) {
	var (
		point repository.ObservationPointWithLikes
		err   error
	)

	idParam := ctx.Param("id")

	if idParam == "" {
		point, err = h.Repository.FirstPublishedObservationPoint()
	} else {
		id, convErr := strconv.Atoi(idParam)
		if convErr != nil || id <= 0 {
			ctx.HTML(http.StatusBadRequest, "feed.page.html", gin.H{
				"point":  nil,
				"active": "feed",
				"error":  "Неверный адрес: идентификатор точки должен быть целым числом",
			})
			return
		}

		if ctx.Query("next") == "true" {
			point, err = h.Repository.NextPublishedObservationPoint(uint(id))
		} else {
			point, err = h.Repository.PublishedObservationPointByID(uint(id))
		}
	}

	switch {
	case errors.Is(err, repository.ErrNotFound):
		ctx.HTML(http.StatusNotFound, "feed.page.html", gin.H{
			"point":  nil,
			"active": "feed",
			"error":  "Точка наблюдения не найдена",
		})
		return
	case err != nil:
		logrus.Errorf("лента: %v", err)
		ctx.HTML(http.StatusInternalServerError, "feed.page.html", gin.H{
			"point":  nil,
			"active": "feed",
			"error":  "Ошибка обращения к базе данных",
		})
		return
	}

	ctx.HTML(http.StatusOK, "feed.page.html", gin.H{
		"point":  h.observationPointView(point),
		"active": "feed",
	})
}

func (h *Handler) GetObservationPoints(ctx *gin.Context) {
	minLatitude, filterError := parseMinLatitude(ctx.Query("min_latitude"))

	points, err := h.Repository.PublishedObservationPoints(minLatitude)
	if err != nil {
		logrus.Errorf("список точек: %v", err)
		filterError = "Ошибка обращения к базе данных"
	}

	views := make([]ObservationPointView, 0, len(points))
	for _, p := range points {
		views = append(views, h.observationPointView(p))
	}

	fill := (minLatitude - minLatitudeLimit) / (maxLatitudeLimit - minLatitudeLimit) * 100

	ctx.HTML(http.StatusOK, "observation_points.page.html", gin.H{
		"points":       views,
		"min_latitude": strconv.FormatFloat(minLatitude, 'f', 0, 64),
		"fill":         strconv.FormatFloat(fill, 'f', 2, 64),
		"active":       "observation_points",
		"error":        filterError,
	})
}

func parseMinLatitude(query string) (float64, string) {
	query = strings.TrimSpace(query)
	if query == "" {
		return minLatitudeLimit, ""
	}

	value, err := strconv.ParseFloat(query, 64)
	switch {
	case err != nil:
		return minLatitudeLimit, "Широта должна быть числом, например 55"
	case value < minLatitudeLimit || value > maxLatitudeLimit:
		return minLatitudeLimit, "Широта лежит в пределах от −90° до 90°"
	}

	return value, ""
}

func (h *Handler) GetObservationPointDraft(ctx *gin.Context) {
	h.renderObservationPointDraft(ctx, http.StatusOK, "")
}

// Черновика нет — шаблон показывает форму создания, это не ошибка.
func (h *Handler) renderObservationPointDraft(ctx *gin.Context, status int, message string) {
	point, err := h.Repository.DraftObservationPoint(CurrentUserID)

	switch {
	case errors.Is(err, repository.ErrNotFound):
		ctx.HTML(status, "add.page.html", gin.H{
			"point":  nil,
			"image":  defaultImageURL,
			"video":  defaultVideoURL,
			"active": "add",
			"error":  message,
		})
		return
	case err != nil:
		logrus.Errorf("черновик: %v", err)
		ctx.HTML(http.StatusInternalServerError, "add.page.html", gin.H{
			"point":  nil,
			"image":  defaultImageURL,
			"video":  defaultVideoURL,
			"active": "add",
			"error":  "Ошибка обращения к базе данных",
		})
		return
	}

	ctx.HTML(status, "add.page.html", gin.H{
		"point":  h.observationPointView(point),
		"active": "add",
		"error":  message,
	})
}

func (h *Handler) CreateObservationPointDraft(ctx *gin.Context) {
	name := strings.TrimSpace(ctx.PostForm("name"))
	if name == "" {
		h.renderObservationPointDraft(ctx, http.StatusBadRequest, "Введите название точки наблюдения")
		return
	}

	description := strings.TrimSpace(ctx.PostForm("description"))
	if description == "" {
		h.renderObservationPointDraft(ctx, http.StatusBadRequest, "Добавьте краткое описание")
		return
	}

	if err := h.Repository.CreateObservationPointDraft(CurrentUserID, name, description); err != nil {
		logrus.Errorf("создание черновика: %v", err)
		h.renderObservationPointDraft(ctx, http.StatusInternalServerError,
			"Не удалось создать черновик: возможно, он уже есть")
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/add")
}

func (h *Handler) PublishObservationPoint(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.PostForm("observation_point_id"))
	if err != nil || id <= 0 {
		h.renderObservationPointDraft(ctx, http.StatusBadRequest, "Неверный идентификатор точки")
		return
	}

	latitude, err := parseCoordinate(ctx.PostForm("latitude"), minLatitudeLimit, maxLatitudeLimit)
	if err != nil {
		h.renderObservationPointDraft(ctx, http.StatusBadRequest, "Широта лежит в пределах от −90° до 90°")
		return
	}

	longitude, err := parseCoordinate(ctx.PostForm("longitude"), minLongitudeLimit, maxLongitudeLimit)
	if err != nil {
		h.renderObservationPointDraft(ctx, http.StatusBadRequest, "Долгота лежит в пределах от −180° до 180°")
		return
	}

	err = h.Repository.PublishObservationPoint(uint(id), latitude, longitude)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		h.renderObservationPointDraft(ctx, http.StatusNotFound, "Черновик не найден")
		return
	case err != nil:
		logrus.Errorf("публикация точки: %v", err)
		h.renderObservationPointDraft(ctx, http.StatusInternalServerError, "Ошибка обращения к базе данных")
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/observation_points")
}

// Координата принимается и с запятой вместо точки.
func parseCoordinate(raw string, min, max float64) (float64, error) {
	raw = strings.ReplaceAll(strings.TrimSpace(raw), ",", ".")

	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, err
	}
	if value < min || value > max {
		return 0, strconv.ErrRange
	}

	return value, nil
}

// Логическое удаление: SQL-запрос без ORM.
func (h *Handler) DeleteObservationPoint(ctx *gin.Context) {
	// считываем значение из формы, которую мы добавили в шаблон
	id, err := strconv.Atoi(ctx.PostForm("observation_point_id"))
	if err != nil || id <= 0 {
		h.errorHandler(ctx, http.StatusBadRequest, errors.New("неверный идентификатор точки"))
		return
	}

	if err := h.Repository.DeleteObservationPoint(uint(id)); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	// Возвращаемся к тому же отфильтрованному списку, из которого пришли.
	target := "/observation_points"
	if minLatitude := ctx.PostForm("min_latitude"); minLatitude != "" {
		target += "?min_latitude=" + minLatitude
	}

	ctx.Redirect(http.StatusSeeOther, target)
}
