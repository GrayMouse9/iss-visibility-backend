package handler

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"iss-visibility/internal/app/repository"
)

const (
	minLatitudeLimit = -90.0
	maxLatitudeLimit = 90.0
)

type Handler struct {
	Repository *repository.Repository
}

func NewHandler(r *repository.Repository) *Handler {
	return &Handler{Repository: r}
}

type PointView struct {
	repository.ObservationPoint
	Likes    int
	ImageURL string
	VideoURL string

	LatAbs       float64
	LatHemi      string
	LatHemiShort string
	LonAbs       float64
	LonHemi      string
	LonHemiShort string
}

func minioBaseURL() string {
	if url := os.Getenv("MINIO_PUBLIC_URL"); url != "" {
		return url
	}
	return "http://localhost:9100/iss-visibility"
}

func view(p repository.ObservationPoint) PointView {
	base := minioBaseURL()

	lat, latHemi, latShort := p.Latitude, "северная", "с. ш."
	if lat < 0 {
		lat, latHemi, latShort = -lat, "южная", "ю. ш."
	}
	lon, lonHemi, lonShort := p.Longitude, "восточная", "в. д."
	if lon < 0 {
		lon, lonHemi, lonShort = -lon, "западная", "з. д."
	}

	return PointView{
		ObservationPoint: p,
		Likes:            len(p.LikedBy),
		ImageURL:         base + "/" + p.ImageKey,
		VideoURL:         base + "/" + p.VideoKey,
		LatAbs:           lat,
		LatHemi:          latHemi,
		LatHemiShort:     latShort,
		LonAbs:           lon,
		LonHemi:          lonHemi,
		LonHemiShort:     lonShort,
	}
}

func (h *Handler) GetFeed(c *gin.Context) {
	var (
		point repository.ObservationPoint
		found bool
	)

	idParam := c.Param("id")

	if idParam == "" {

		point, found = h.Repository.FirstPublished()
	} else {
		id, err := strconv.Atoi(idParam)
		if err != nil {
			c.HTML(http.StatusBadRequest, "feed.html", gin.H{
				"point":  nil,
				"active": "feed",
				"error":  "Неверный адрес: идентификатор точки должен быть целым числом",
			})
			return
		}

		if c.Query("next") == "true" {
			point, found = h.Repository.NextPublished(id)
		} else {
			point, found = h.Repository.PublishedByID(id)
		}
	}

	if !found {
		c.HTML(http.StatusNotFound, "feed.html", gin.H{
			"point":  nil,
			"active": "feed",
			"error":  "Точка наблюдения не найдена",
		})
		return
	}

	c.HTML(http.StatusOK, "feed.html", gin.H{
		"point":  view(point),
		"active": "feed",
	})
}

func (h *Handler) GetDraft(c *gin.Context) {
	point, found := h.Repository.Draft()
	if !found {
		c.HTML(http.StatusNotFound, "add.html", gin.H{
			"point":  nil,
			"active": "add",
			"error":  "В коллекции нет точки со статусом «черновик»",
		})
		return
	}

	c.HTML(http.StatusOK, "add.html", gin.H{
		"point":  view(point),
		"active": "add",
	})
}

func (h *Handler) GetObservationPoints(c *gin.Context) {
	minLatitudeQuery := strings.TrimSpace(c.Query("min_latitude"))

	minLatitude := minLatitudeLimit
	filterError := ""

	if minLatitudeQuery != "" {
		value, err := strconv.ParseFloat(minLatitudeQuery, 64)
		switch {
		case err != nil:
			filterError = "Широта должна быть целым числом, например 55"
		case value < minLatitudeLimit || value > maxLatitudeLimit:
			filterError = "Широта лежит в пределах от −90° до 90°"
		default:
			minLatitude = value
		}
	}

	points := h.Repository.PublishedByMinLatitude(minLatitude)

	views := make([]PointView, 0, len(points))
	for _, p := range points {
		views = append(views, view(p))
	}

	fill := (minLatitude - minLatitudeLimit) / (maxLatitudeLimit - minLatitudeLimit) * 100

	c.HTML(http.StatusOK, "observation_points.html", gin.H{
		"points":       views,
		"min_latitude": minLatitude,
		"fill":         fill,
		"active":       "observation_points",
		"error":        filterError,
	})
}
