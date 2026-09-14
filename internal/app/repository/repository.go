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

	ImageKey string
	VideoKey string

	LikedBy []int
}

type Repository struct {
	observationPoints []ObservationPoint
}

func NewRepository() (*Repository, error) {
	return &Repository{observationPoints: observationPoints()}, nil
}

func (r *Repository) Published() []ObservationPoint {
	result := make([]ObservationPoint, 0, len(r.observationPoints))
	for _, point := range r.observationPoints {
		if point.Status == StatusPublished {
			result = append(result, point)
		}
	}
	return result
}

func (r *Repository) PublishedByMinLatitude(minLatitude float64) []ObservationPoint {
	result := make([]ObservationPoint, 0, len(r.observationPoints))
	for _, point := range r.Published() {
		if point.Latitude >= minLatitude {
			result = append(result, point)
		}
	}
	return result
}

func (r *Repository) PublishedByID(id int) (ObservationPoint, bool) {
	for _, point := range r.Published() {
		if point.ID == id {
			return point, true
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
	for i, point := range published {
		if point.ID == id {
			return published[(i+1)%len(published)], true
		}
	}
	return ObservationPoint{}, false

}

func (r *Repository) Draft() (ObservationPoint, bool) {
	for _, point := range r.observationPoints {
		if point.Status == StatusDraft {
			return point, true
		}
	}
	return ObservationPoint{}, false
}

func observationPoints() []ObservationPoint {
	return []ObservationPoint{
		{
			ID:          1,
			Name:        "Обсерватория Джемини-Север, Мауна-Кеа",
			Description: "Восьмиметровый телескоп на вершине вулкана, 4200 м над уровнем моря. Выше обсерватории остаётся лишь 40 % воздуха, и он почти не искажает изображение.",
			Status:      StatusPublished,
			Latitude:    19.8207,
			Longitude:   -155.4681,
			ImageKey:    "maunakea.jpg",
			VideoKey:    "maunakea.mp4",
			LikedBy:     []int{2, 5, 9, 14, 21, 30, 38},
		},
		{
			ID:          2,
			Name:        "Национальная обсерватория Китт-Пик, Аризона",
			Description: "Более двадцати телескопов на одной горе в пустыне Сонора. Ясных ночей здесь около 260 в году, что делает площадку удобной для регулярных наблюдений.",
			Status:      StatusPublished,
			Latitude:    31.9583,
			Longitude:   -111.5967,
			ImageKey:    "kittpeak.jpg",
			VideoKey:    "kittpeak.mp4",
			LikedBy:     []int{1, 6, 13, 22, 27},
		},
		{
			ID:          3,
			Name:        "Обсерватория ALMA, плато Чахнантор",
			Description: "Шестьдесят шесть антенн на высоте 5000 м — самая высокогорная обсерватория мира. Воздух предельно сух, помех для наблюдения практически нет.",
			Status:      StatusPublished,
			Latitude:    -23.0294,
			Longitude:   -67.7548,
			ImageKey:    "alma.jpg",
			VideoKey:    "alma.mp4",
			LikedBy:     []int{3, 8, 11, 19, 25, 34, 40},
		},
		{
			ID:          4,
			Name:        "Паранальская обсерватория, Атакама",
			Description: "Четыре восьмиметровых телескопа в пустыне Атакама. Осадки здесь почти не выпадают, а ясных ночей больше трёхсот в году.",
			Status:      StatusPublished,
			Latitude:    -24.6275,
			Longitude:   -70.4044,
			ImageKey:    "paranal.jpg",
			VideoKey:    "paranal.mp4",
			LikedBy:     []int{4, 10, 17, 23, 29, 36},
		},
		{
			ID:          5,
			Name:        "Обсерватория Ла-Силья, Чили",
			Description: "Первая площадка Южной европейской обсерватории, работает с 1969 года. До ближайшего города сотни километров, поэтому небо здесь остаётся полностью тёмным.",
			Status:      StatusPublished,
			Latitude:    -29.2567,
			Longitude:   -70.7300,
			ImageKey:    "lasilla.jpg",
			VideoKey:    "lasilla.mp4",
			LikedBy:     []int{7, 12, 20, 31},
		},
		{
			ID:          6,
			Name:        "Байконур, космодром",
			Description: "Крупнейший космодром мира, стартовая площадка пилотируемых полётов к МКС. Ровная степь открывает обзор до самого горизонта во все стороны.",
			Status:      StatusDraft,
			Latitude:    45.9646,
			Longitude:   63.3052,
			ImageKey:    "baikonur.jpg",
			VideoKey:    "baikonur.mp4",
			LikedBy:     []int{},
		},
		{
			ID:          7,
			Name:        "МГТУ им. Н. Э. Баумана, Москва",
			Description: "Учебная площадка в центре Москвы. Городская засветка гасит слабые объекты, но МКС достигает −3,5 звёздной величины и видна невооружённым глазом.",
			Status:      StatusDeleted,
			Latitude:    55.7660,
			Longitude:   37.6849,
			ImageKey:    "bmstu.jpg",
			VideoKey:    "bmstu.mp4",
			LikedBy:     []int{15, 18, 26, 33},
		},
	}
}
