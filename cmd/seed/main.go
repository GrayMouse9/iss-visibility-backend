package main

import (
	"log"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"iss-visibility/internal/app/config"
	"iss-visibility/internal/app/ds"
	"iss-visibility/internal/app/dsn"
)

// Указатель нужен для необязательных полей модели.
func ptr[T any](v T) *T {
	return &v
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("файл .env не найден, беру переменные из окружения")
	}

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("не удалось прочитать конфигурацию: %v", err)
	}
	media := cfg.MinioPublicURL

	db, err := gorm.Open(postgres.Open(dsn.FromEnv()), &gorm.Config{})
	if err != nil {
		log.Fatalf("не удалось подключиться к базе: %v", err)
	}

	// Очистка таблиц, чтобы команду можно было запускать повторно.
	err = db.Exec(`TRUNCATE point_likes, observation_points, users
	               RESTART IDENTITY CASCADE`).Error
	if err != nil {
		log.Fatalf("не удалось очистить таблицы: %v", err)
	}

	users := []ds.User{
		{Login: "dasha", Password: "dasha", IsModerator: false},
		{Login: "moderator", Password: "moderator", IsModerator: true},
		{Login: "orion", Password: "orion"},
		{Login: "vega", Password: "vega"},
		{Login: "lyra", Password: "lyra"},
		{Login: "sirius", Password: "sirius"},
		{Login: "polaris", Password: "polaris"},
		{Login: "antares", Password: "antares"},
	}
	if err := db.Create(&users).Error; err != nil {
		log.Fatalf("не удалось создать пользователей: %v", err)
	}

	created := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	formed := time.Date(2026, 9, 10, 12, 30, 0, 0, time.UTC)

	points := []ds.ObservationPoint{
		{
			Name:        "Обсерватория Джемини-Север, Мауна-Кеа",
			Description: "Восьмиметровый телескоп на вершине вулкана, 4200 м над уровнем моря. Выше обсерватории остаётся лишь 40 % воздуха, и он почти не искажает изображение.",
			Status:      ds.PointPublished,
			ImageURL:    media + "/maunakea.jpg",
			VideoURL:    media + "/maunakea.mp4",
			Latitude:    19.8207,
			Longitude:   -155.4681,
			DateCreated: created,
			DateFormed:  ptr(formed),
			CreatorID:   users[0].ID,
		},
		{
			Name:        "Национальная обсерватория Китт-Пик, Аризона",
			Description: "Более двадцати телескопов на одной горе в пустыне Сонора. Ясных ночей здесь около 260 в году, что делает площадку удобной для регулярных наблюдений.",
			Status:      ds.PointPublished,
			ImageURL:    media + "/kittpeak.jpg",
			VideoURL:    media + "/kittpeak.mp4",
			Latitude:    31.9583,
			Longitude:   -111.5967,
			DateCreated: created,
			DateFormed:  ptr(formed),
			CreatorID:   users[0].ID,
		},
		{
			Name:        "Обсерватория ALMA, плато Чахнантор",
			Description: "Шестьдесят шесть антенн на высоте 5000 м — самая высокогорная обсерватория мира. Воздух предельно сух, помех для наблюдения практически нет.",
			Status:      ds.PointPublished,
			ImageURL:    media + "/alma.jpg",
			VideoURL:    media + "/alma.mp4",
			Latitude:    -23.0294,
			Longitude:   -67.7548,
			DateCreated: created,
			DateFormed:  ptr(formed),
			CreatorID:   users[0].ID,
		},
		{
			Name:        "Паранальская обсерватория, Атакама",
			Description: "Четыре восьмиметровых телескопа в пустыне Атакама. Осадки здесь почти не выпадают, а ясных ночей больше трёхсот в году.",
			Status:      ds.PointPublished,
			ImageURL:    media + "/paranal.jpg",
			VideoURL:    media + "/paranal.mp4",
			Latitude:    -24.6275,
			Longitude:   -70.4044,
			DateCreated: created,
			DateFormed:  ptr(formed),
			CreatorID:   users[0].ID,
		},
		{
			Name:        "Обсерватория Ла-Силья, Чили",
			Description: "Первая площадка Южной европейской обсерватории, работает с 1969 года. До ближайшего города сотни километров, поэтому небо здесь остаётся полностью тёмным.",
			Status:      ds.PointPublished,
			ImageURL:    media + "/lasilla.jpg",
			VideoURL:    media + "/lasilla.mp4",
			Latitude:    -29.2567,
			Longitude:   -70.7300,
			DateCreated: created,
			DateFormed:  ptr(formed),
			CreatorID:   users[0].ID,
		},
		{
			// Ссылки пустые: на страницах подставятся файлы по умолчанию.
			Name:        "МГТУ им. Н. Э. Баумана",
			Status:      ds.PointDraft,
			Latitude:    45.9646,
			Longitude:   63.3052,
			DateCreated: time.Now(),
			CreatorID:   users[0].ID,
		},
		{
			Name:        "МГТУ им. Н. Э. Баумана, Москва",
			Description: "Учебная площадка в центре Москвы. Городская засветка гасит слабые объекты, но МКС достигает −3,5 звёздной величины и видна невооружённым глазом.",
			Status:      ds.PointDeleted,
			ImageURL:    media + "/bmstu.jpg",
			VideoURL:    media + "/bmstu.mp4",
			Latitude:    55.7660,
			Longitude:   37.6849,
			DateCreated: created,
			DateFormed:  ptr(formed),
			CreatorID:   users[0].ID,
		},
	}
	if err := db.Omit("Creator").Create(&points).Error; err != nil {
		log.Fatalf("не удалось создать точки наблюдения: %v", err)
	}

	likeCounts := []int{7, 5, 7, 6, 4, 0, 4}

	var likes []ds.PointLike
	for i, point := range points {
		for j := 0; j < likeCounts[i] && j < len(users); j++ {
			likes = append(likes, ds.PointLike{
				UserID:  users[j].ID,
				PointID: point.ID,
			})
		}
	}
	if err := db.Omit("User", "Point").Create(&likes).Error; err != nil {
		log.Fatalf("не удалось создать лайки: %v", err)
	}

	log.Printf("Готово: пользователей %d, точек %d, лайков %d",
		len(users), len(points), len(likes))
}
