package processorloop

import (
	"time"
)

func doNotification() {

}

func getCurMskTime() (uint64, error) {
	currentTime := time.Now().Unix()

	// loc, err := time.LoadLocation("Europe/Moscow")
	// if err != nil {
	// 	return 0, e.Wrap("Ошибка при загрузке часового пояса Москвы:", err)
	// }

	// // Преобразуем текущее время в московское время
	// moscowTime := time.Unix(currentTime, 0).In(loc)

	// return moscowTime, nil

	return uint64(currentTime), nil
}
