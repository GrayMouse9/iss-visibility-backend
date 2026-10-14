package main

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

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

	media := os.Getenv("MINIO_PUBLIC_URL")

	db, err := gorm.Open(postgres.Open(dsn.FromEnv()), &gorm.Config{})
	if err != nil {
		log.Fatalf("не удалось подключиться к базе: %v", err)
	}

	// Очистка таблиц, чтобы команду можно было запускать повторно.
	err = db.Exec(`TRUNCATE observation_point_likes, observation_points, users
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

	// В поле Password пока лежит открытый пароль — заменяем его на bcrypt-хэш.
	// Открытые пароли остаются только в этом файле, чтобы было чем входить при показе.
	for i := range users {
		hash, err := bcrypt.GenerateFromPassword([]byte(users[i].Password), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("не удалось захэшировать пароль %q: %v", users[i].Login, err)
		}

		users[i].Password = string(hash)
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
			Status:      ds.StatusPublished,
			ImageURL:    media + "/maunakea.jpg",
			VideoURL:    media + "/maunakea.mp4",
			Latitude:    ptr(19.8207),
			Longitude:   ptr(-155.4681),
			DateCreated: created,
			DateFormed:  ptr(formed),
			CreatorID:   users[0].ID,
		},
		{
			Name:        "Национальная обсерватория Китт-Пик, Аризона",
			Description: "Более двадцати телескопов на одной горе в пустыне Сонора. Ясных ночей здесь около 260 в году, что делает площадку удобной для регулярных наблюдений.",
			Status:      ds.StatusPublished,
			ImageURL:    media + "/kittpeak.jpg",
			VideoURL:    media + "/kittpeak.mp4",
			Latitude:    ptr(31.9583),
			Longitude:   ptr(-111.5967),
			DateCreated: created,
			DateFormed:  ptr(formed),
			CreatorID:   users[0].ID,
		},
		{
			// Удалена намеренно: создаёт пропуск в середине, на нём видно работу кнопки «Далее».
			Name:        "МГТУ им. Н. Э. Баумана, Москва",
			Description: "Учебная площадка в центре Москвы. Городская засветка гасит слабые объекты, но МКС достигает −3,5 звёздной величины и видна невооружённым глазом.",
			Status:      ds.StatusDeleted,
			ImageURL:    media + "/bmstu.jpg",
			VideoURL:    media + "/bmstu.mp4",
			Latitude:    ptr(55.7660),
			Longitude:   ptr(37.6849),
			DateCreated: created,
			DateFormed:  ptr(formed),
			CreatorID:   users[0].ID,
		},
		{
			Name:        "Обсерватория ALMA, плато Чахнантор",
			Description: "Шестьдесят шесть антенн на высоте 5000 м — самая высокогорная обсерватория мира. Воздух предельно сух, помех для наблюдения практически нет.",
			Status:      ds.StatusPublished,
			ImageURL:    media + "/alma.jpg",
			VideoURL:    media + "/alma.mp4",
			Latitude:    ptr(-23.0294),
			Longitude:   ptr(-67.7548),
			DateCreated: created,
			DateFormed:  ptr(formed),
			CreatorID:   users[0].ID,
		},
		{
			Name:        "Паранальская обсерватория, Атакама",
			Description: "Четыре восьмиметровых телескопа в пустыне Атакама. Осадки здесь почти не выпадают, а ясных ночей больше трёхсот в году.",
			Status:      ds.StatusPublished,
			ImageURL:    media + "/paranal.jpg",
			VideoURL:    media + "/paranal.mp4",
			Latitude:    ptr(-24.6275),
			Longitude:   ptr(-70.4044),
			DateCreated: created,
			DateFormed:  ptr(formed),
			CreatorID:   users[0].ID,
		},
		{
			Name:        "Обсерватория Ла-Силья, Чили",
			Description: "Первая площадка Южной европейской обсерватории, работает с 1969 года. До ближайшего города сотни километров, поэтому небо здесь остаётся полностью тёмным.",
			Status:      ds.StatusPublished,
			ImageURL:    media + "/lasilla.jpg",
			VideoURL:    media + "/lasilla.mp4",
			Latitude:    ptr(-29.2567),
			Longitude:   ptr(-70.7300),
			DateCreated: created,
			DateFormed:  ptr(formed),
			CreatorID:   users[0].ID,
		},
		{
			// Черновик другого пользователя: у текущего место свободно, его черновик
			// создаётся запросом. Ссылки и координаты пустые — они появляются позже.
			Name:        "МГТУ им. Н. Э. Баумана",
			Status:      ds.StatusDraft,
			DateCreated: time.Now(),
			CreatorID:   users[2].ID,
		},
	}
	if err := db.Omit("Creator").Create(&points).Error; err != nil {
		log.Fatalf("не удалось создать точки наблюдения: %v", err)
	}

	likeCounts := []int{7, 5, 4, 7, 6, 4, 0}

	var likes []ds.ObservationPointLike
	for i, point := range points {
		for j := 0; j < likeCounts[i] && j < len(users); j++ {
			likes = append(likes, ds.ObservationPointLike{
				UserID:             users[j].ID,
				ObservationPointID: point.ID,
			})
		}
	}
	if err := db.Omit("User", "ObservationPoint").Create(&likes).Error; err != nil {
		log.Fatalf("не удалось создать лайки: %v", err)
	}

	log.Printf("Готово: пользователей %d, точек %d, лайков %d",
		len(users), len(points), len(likes))
}
