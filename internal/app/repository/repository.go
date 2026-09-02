package repository

type Status string

const (
	StatusDraft     Status = "черновик"
	StatusPublished Status = "опубликован"
	StatusDeleted   Status = "удален"
)

type ObservationPoint struct {
	ID          int
	Name        string
	Description string
	Status      Status

	Latitude  float64
	Longitude float64
	Altitude  int

	ImageKey string
	VideoKey string

	LikedBy []int
}

type Repository struct {
	points []ObservationPoint
}

func NewRepository() (*Repository, error) {
	return &Repository{points: collection()}, nil
}

func (r *Repository) Published() []ObservationPoint {
	result := make([]ObservationPoint, 0, len(r.points))
	for _, p := range r.points {
		if p.Status == StatusPublished {
			result = append(result, p)
		}
	}
	return result
}

func (r *Repository) PublishedByMinAltitude(minAltitude int) []ObservationPoint {
	result := make([]ObservationPoint, 0, len(r.points))
	for _, p := range r.Published() {
		if p.Altitude >= minAltitude {
			result = append(result, p)
		}
	}
	return result
}

func (r *Repository) PublishedByID(id int) (ObservationPoint, bool) {
	for _, p := range r.Published() {
		if p.ID == id {
			return p, true
		}
	}
	return ObservationPoint{}, false
}

func (r *Repository) FirstPublished() (ObservationPoint, bool) {
	published := r.Published()
	if len(published) == 0 {
		return ObservationPoint{}, false
	}
	return published[0], true
}

func (r *Repository) NextPublished(id int) (ObservationPoint, bool) {
	published := r.Published()
	if len(published) == 0 {
		return ObservationPoint{}, false
	}
	for i, p := range published {
		if p.ID == id {
			return published[(i+1)%len(published)], true
		}
	}
	return published[0], true
}

func (r *Repository) Draft() (ObservationPoint, bool) {
	for _, p := range r.points {
		if p.Status == StatusDraft {
			return p, true
		}
	}
	return ObservationPoint{}, false
}

func collection() []ObservationPoint {
	return []ObservationPoint{
		{
			ID:          1,
			Name:        "МГТУ им. Н.Э. Баумана, Москва",
			Description: "Учебная площадка наблюдения в центре Москвы. Световое загрязнение ограничивает видимость слабых объектов, но МКС при благоприятном пролёте достигает −3,5 звёздной величины и уверенно видна невооружённым глазом.",
			Status:      StatusPublished,
			Latitude:    55.7660,
			Longitude:   37.6849,
			Altitude:    150,
			ImageKey:    "bmstu.jpg",
			VideoKey:    "bmstu.mp4",
			LikedBy:     userIDs(128),
		},
		{
			ID:          2,
			Name:        "Пулковская обсерватория, Санкт-Петербург",
			Description: "Старейшая астрономическая обсерватория России, основана в 1839 году. На широте 59,8° летом наступают белые ночи, поэтому окно наблюдения МКС смещается.",
			Status:      StatusPublished,
			Latitude:    59.7714,
			Longitude:   30.3264,
			Altitude:    75,
			ImageKey:    "pulkovo.jpg",
			VideoKey:    "pulkovo.mp4",
			LikedBy:     userIDs(64),
		},
		{
			ID:          9,
			Name:        "Комплекс дальней космической связи Голдстоун, Калифорния",
			Description: "Станция дальней космической связи NASA в пустыне Мохаве на высоте 900 м. Сухой воздух и удалённость от городов дают почти полное отсутствие засветки, поэтому пролёт МКС прослеживается здесь от горизонта до горизонта.",
			Status:      StatusPublished,
			Latitude:    35.4267,
			Longitude:   -116.8900,
			Altitude:    900,
			ImageKey:    "goldstone.jpg",
			VideoKey:    "goldstone.mp4",
			LikedBy:     userIDs(128),
		},
		{
			ID:          7,
			Name:        "Байконур, космодром",
			Description: "Крупнейший космодром мира, стартовая площадка пилотируемых полётов к МКС.",
			Status:      StatusDraft,
			Latitude:    45.9646,
			Longitude:   63.3052,
			Altitude:    90,
			ImageKey:    "baikonur.jpg",
			VideoKey:    "baikonur.mp4",
			LikedBy:     nil,
		},
		{
			ID:          8,
			Name:        "Байкальская астрофизическая обсерватория, Листвянка",
			Description: "Солнечный телескоп на берегу Байкала.",
			Status:      StatusDeleted,
			Latitude:    51.8464,
			Longitude:   104.8919,
			Altitude:    680,
			ImageKey:    "baikal.jpg",
			VideoKey:    "baikal.mp4",
			LikedBy:     userIDs(17),
		},
	}
}

func userIDs(count int) []int {
	ids := make([]int, count)
	for i := range ids {
		ids[i] = i + 1
	}
	return ids
}
