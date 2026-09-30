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
type PointView struct {
	repository.PointWithLikes
	ImageURL string
	VideoURL string

	LatAbs       float64
	LatHemi      string
	LatHemiShort string
	LonAbs       float64
	LonHemi      string
	LonHemiShort string
}

func (h *Handler) view(p repository.PointWithLikes) PointView {
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

	return PointView{
		PointWithLikes: p,
		ImageURL:       image,
		VideoURL:       video,
		LatAbs:         lat,
		LatHemi:        latHemi,
		LatHemiShort:   latShort,
		LonAbs:         lon,
		LonHemi:        lonHemi,
		LonHemiShort:   lonShort,
	}
}

func (h *Handler) GetFeed(ctx *gin.Context) {
	var (
		point repository.PointWithLikes
		err   error
	)

	idParam := ctx.Param("id")

	if idParam == "" {
		point, err = h.Repository.FirstPublishedPoint()
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
			point, err = h.Repository.NextPublishedPoint(uint(id))
		} else {
			point, err = h.Repository.PublishedPointByID(uint(id))
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
		"point":  h.view(point),
		"active": "feed",
	})
}

func (h *Handler) GetObservationPoints(ctx *gin.Context) {
	minLatitude, filterError := parseMinLatitude(ctx.Query("min_latitude"))

	points, err := h.Repository.PublishedPoints(minLatitude)
	if err != nil {
		logrus.Errorf("список точек: %v", err)
		filterError = "Ошибка обращения к базе данных"
	}

	views := make([]PointView, 0, len(points))
	for _, p := range points {
		views = append(views, h.view(p))
	}

	fill := (minLatitude - minLatitudeLimit) / (maxLatitudeLimit - minLatitudeLimit) * 100

	ctx.HTML(http.StatusOK, "observation_points.page.html", gin.H{
		"points":       views,
		"min_latitude": minLatitude,
		"fill":         fill,
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

func (h *Handler) GetDraft(ctx *gin.Context) {
	h.renderDraft(ctx, http.StatusOK, "")
}

// Черновика нет — шаблон показывает форму создания, это не ошибка.
func (h *Handler) renderDraft(ctx *gin.Context, status int, message string) {
	point, err := h.Repository.DraftPoint(CurrentUserID)

	switch {
	case errors.Is(err, repository.ErrNotFound):
		ctx.HTML(status, "add.page.html", gin.H{
			"point":  nil,
			"active": "add",
			"error":  message,
		})
		return
	case err != nil:
		logrus.Errorf("черновик: %v", err)
		ctx.HTML(http.StatusInternalServerError, "add.page.html", gin.H{
			"point":  nil,
			"active": "add",
			"error":  "Ошибка обращения к базе данных",
		})
		return
	}

	ctx.HTML(status, "add.page.html", gin.H{
		"point":  h.view(point),
		"active": "add",
		"error":  message,
	})
}

func (h *Handler) CreateDraft(ctx *gin.Context) {
	name := strings.TrimSpace(ctx.PostForm("name"))
	if name == "" {
		h.renderDraft(ctx, http.StatusBadRequest, "Введите название точки наблюдения")
		return
	}

	if err := h.Repository.CreateDraft(CurrentUserID, name); err != nil {
		logrus.Errorf("создание черновика: %v", err)
		h.renderDraft(ctx, http.StatusInternalServerError,
			"Не удалось создать черновик: возможно, он уже есть")
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/add")
}

func (h *Handler) PublishPoint(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.PostForm("point_id"))
	if err != nil || id <= 0 {
		h.renderDraft(ctx, http.StatusBadRequest, "Неверный идентификатор точки")
		return
	}

	description := strings.TrimSpace(ctx.PostForm("description"))
	if description == "" {
		h.renderDraft(ctx, http.StatusBadRequest, "Добавьте краткое описание")
		return
	}

	latitude, err := parseCoordinate(ctx.PostForm("latitude"), minLatitudeLimit, maxLatitudeLimit)
	if err != nil {
		h.renderDraft(ctx, http.StatusBadRequest, "Широта лежит в пределах от −90° до 90°")
		return
	}

	longitude, err := parseCoordinate(ctx.PostForm("longitude"), minLongitudeLimit, maxLongitudeLimit)
	if err != nil {
		h.renderDraft(ctx, http.StatusBadRequest, "Долгота лежит в пределах от −180° до 180°")
		return
	}

	err = h.Repository.PublishPoint(uint(id), description, latitude, longitude)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		h.renderDraft(ctx, http.StatusNotFound, "Черновик не найден")
		return
	case err != nil:
		logrus.Errorf("публикация точки: %v", err)
		h.renderDraft(ctx, http.StatusInternalServerError, "Ошибка обращения к базе данных")
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
func (h *Handler) DeletePoint(ctx *gin.Context) {
	// считываем значение из формы, которую мы добавили в шаблон
	id, err := strconv.Atoi(ctx.PostForm("point_id"))
	if err != nil || id <= 0 {
		h.errorHandler(ctx, http.StatusBadRequest, errors.New("неверный идентификатор точки"))
		return
	}

	if err := h.Repository.DeletePoint(uint(id)); err != nil {
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
