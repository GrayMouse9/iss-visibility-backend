package handler

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"iss-visibility/internal/app/repository"
)

const maxAltitude = 8849

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

	LatAbs  float64
	LatHemi string
	LonAbs  float64
	LonHemi string
}

func minioBaseURL() string {
	if url := os.Getenv("MINIO_PUBLIC_URL"); url != "" {
		return url
	}
	return "http://localhost:9100/iss-visibility"
}

func view(p repository.ObservationPoint) PointView {
	base := minioBaseURL()

	lat, latHemi := p.Latitude, "N"
	if lat < 0 {
		lat, latHemi = -lat, "S"
	}
	lon, lonHemi := p.Longitude, "E"
	if lon < 0 {
		lon, lonHemi = -lon, "W"
	}

	return PointView{
		ObservationPoint: p,
		Likes:            len(p.LikedBy),
		ImageURL:         base + "/" + p.ImageKey,
		VideoURL:         base + "/" + p.VideoKey,
		LatAbs:           lat,
		LatHemi:          latHemi,
		LonAbs:           lon,
		LonHemi:          lonHemi,
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

func (h *Handler) GetPoints(c *gin.Context) {
	minAltitudeQuery := strings.TrimSpace(c.Query("min_altitude"))

	minAltitude := 0
	filterError := ""

	if minAltitudeQuery != "" {
		value, err := strconv.Atoi(minAltitudeQuery)
		switch {
		case err != nil:
			filterError = "Высота должна быть целым числом, например 900"
		case value < 0:
			filterError = "Высота над уровнем моря не может быть отрицательной"
		case value > maxAltitude:
			filterError = "Слишком большое значение: высочайшая точка суши — 8849 м"
		default:
			minAltitude = value
		}
	}

	points := h.Repository.PublishedByMinAltitude(minAltitude)

	views := make([]PointView, 0, len(points))
	for _, p := range points {
		views = append(views, view(p))
	}

	c.HTML(http.StatusOK, "points.html", gin.H{
		"points": views,

		"min_altitude": minAltitudeQuery,
		"active":       "points",
		"error":        filterError,
	})
}
