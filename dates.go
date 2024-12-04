package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// checkTaskData проверяет и обрабатывает данные задачи
func checkTaskData(task *Task, now time.Time) error {
	// Нормализуем текущую дату до формата 20060102
	now, _ = time.Parse("20060102", now.Format("20060102"))

	// Если поле "date" пустое, подставляем текущую дату
	if task.Date == "" {
		task.Date = now.Format("20060102")
	}

	// Пробуем распарсить дату
	parsedDate, err := time.Parse("20060102", task.Date)
	if err != nil {
		return fmt.Errorf(ErrorTaskNotFound)
	}

	// Если дата меньше текущей и repeat пустое, обновляем дату на текущую
	if parsedDate.Before(now) && task.Repeat == "" {
		task.Date = now.Format("20060102")
	} else if parsedDate.Before(now) && task.Repeat != "" {
		// Если дата меньше текущей и есть repeat, вычисляем следующую дату
		task.Date, err = NextDate(now, task.Date, task.Repeat)
		if err != nil {
			return fmt.Errorf(ErrorRepeatRule)
		}
	}

	return nil
}

// NextDate вычисляет следующую дату для задачи в соответствии с правилом повторения.
func NextDate(now time.Time, date string, repeat string) (string, error) {
	repeat = strings.TrimSpace(repeat)
	if repeat == "" {
		return "", fmt.Errorf("правило повторения не указано")
	}

	dateTime, err := time.Parse("20060102", date)
	if err != nil {
		return "", fmt.Errorf("некорректный формат даты: %v", err)
	}

	// Определяем тип правила и вызываем соответствующую функцию.
	switch {
	case strings.HasPrefix(repeat, "d "):
		return calculateNextDateDaily(now, dateTime, repeat)
	case repeat == "y":
		return calculateNextDateYearly(now, dateTime)
	case strings.HasPrefix(repeat, "w "):
		return calculateNextDateWeekly(now, dateTime, repeat)
	case strings.HasPrefix(repeat, "m "):
		return calculateNextDateMonthly(now, dateTime, repeat)
	default:
		return "", fmt.Errorf("неподдерживаемый формат правила повторения")
	}
}

// calculateNextDateDaily обрабатывает правило повторения вида 'd <число>'.
func calculateNextDateDaily(now time.Time, dateTime time.Time, repeat string) (string, error) {
	daysStr := strings.TrimSpace(repeat[2:])
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 || days > 400 {
		return "", fmt.Errorf("некорректный интервал в днях")
	}

	// Продвигаем дату минимум на один интервал вперёд.
	nextDate := dateTime.AddDate(0, 0, days)
	for !nextDate.After(now) {
		nextDate = nextDate.AddDate(0, 0, days)
	}
	return nextDate.Format("20060102"), nil
}

// calculateNextDateYearly обрабатывает правило повторения 'y'.
func calculateNextDateYearly(now time.Time, dateTime time.Time) (string, error) {
	nextDate := dateTime.AddDate(1, 0, 0)
	for !nextDate.After(now) {
		nextDate = nextDate.AddDate(1, 0, 0)
	}

	// Обработка високосного года для 29 февраля.
	if dateTime.Month() == time.February && dateTime.Day() == 29 {
		if nextDate.Month() == time.February && nextDate.Day() == 28 {
			nextDate = nextDate.AddDate(0, 0, 1)
		}
	}

	return nextDate.Format("20060102"), nil
}

// calculateNextDateWeekly обрабатывает правило повторения вида 'w <дни недели>'.
func calculateNextDateWeekly(now time.Time, dateTime time.Time, repeat string) (string, error) {
	// Извлекаем список дней недели из правила повторения.
	daysStr := strings.TrimSpace(repeat[2:])
	dayStrings := strings.Split(daysStr, ",")
	if len(dayStrings) == 0 {
		return "", fmt.Errorf(ErrorRepeatRule)
	}

	// Преобразуем дни недели в числа и проверяем их корректность.
	targetWeekdays := make([]time.Weekday, 0, len(dayStrings))
	for _, s := range dayStrings {
		dayNum, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil || dayNum < 1 || dayNum > 7 {
			return "", fmt.Errorf("некорректный день недели: %s", s)
		}
		// Преобразуем день недели в time.Weekday (понедельник — 1, воскресенье — 7)
		var weekday time.Weekday
		if dayNum == 7 {
			weekday = time.Sunday
		} else {
			weekday = time.Weekday(dayNum)
		}
		targetWeekdays = append(targetWeekdays, weekday)
	}

	// Инициализируем переменную для хранения ближайшей даты.
	var earliestDate time.Time

	// Текущий день недели согласно нумерации Go (0=Воскресенье, ..., 6=Суббота).
	currentWeekday := now.Weekday()

	// Ищем ближайшую дату после now и dateTime из указанных дней недели.
	for _, targetWeekday := range targetWeekdays {
		// Вычисляем количество дней до следующего указанного дня недели.
		daysUntil := (int(targetWeekday) - int(currentWeekday) + 7) % 7
		if daysUntil == 0 {
			daysUntil = 7 // Если сегодня, переносим на следующую неделю
		}
		candidateDate := now.AddDate(0, 0, daysUntil)
		// Проверяем, что кандидат после исходной даты.
		if candidateDate.Before(dateTime) {
			candidateDate = candidateDate.AddDate(0, 0, 7)
		}
		// Находим самую раннюю дату.
		if earliestDate.IsZero() || candidateDate.Before(earliestDate) {
			earliestDate = candidateDate
		}
	}

	if earliestDate.IsZero() {
		return "", fmt.Errorf("не удалось найти дату повторения")
	}

	return earliestDate.Format("20060102"), nil
}

// calculateNextDateMonthly обрабатывает правило повторения вида 'm <дни> [<месяцы>]'.
func calculateNextDateMonthly(now time.Time, dateTime time.Time, repeat string) (string, error) {
	// Извлекаем часть правила после 'm '.
	ruleContent := strings.TrimSpace(repeat[2:])
	if ruleContent == "" {
		return "", fmt.Errorf("некорректное правило повторения: отсутствуют дни")
	}

	// Разделяем дни и месяцы, если они указаны.
	parts := strings.SplitN(ruleContent, " ", 2)
	daysPart := parts[0]
	monthsPart := ""
	if len(parts) > 1 {
		monthsPart = parts[1]
	}

	// Парсим дни месяца.
	dayStrings := strings.Split(daysPart, ",")
	if len(dayStrings) == 0 {
		return "", fmt.Errorf("некорректное правило повторения: отсутствуют дни")
	}

	days := make([]int, 0, len(dayStrings))
	for _, s := range dayStrings {
		day, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil || ((day < 1 || day > 31) && day != -1 && day != -2) {
			return "", fmt.Errorf("некорректный день месяца: %s", s)
		}
		days = append(days, day)
	}

	// Парсим месяцы, если они указаны.
	months := make([]time.Month, 0)
	if monthsPart != "" {
		monthStrings := strings.Split(monthsPart, ",")
		for _, s := range monthStrings {
			monthNum, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil || monthNum < 1 || monthNum > 12 {
				return "", fmt.Errorf("некорректный номер месяца: %s", s)
			}
			months = append(months, time.Month(monthNum))
		}
	}

	// Функция для получения последнего дня месяца.
	lastDayOfMonth := func(year int, month time.Month) int {
		// Переходим на первый день следующего месяца и вычитаем один день.
		t := time.Date(year, month+1, 1, 0, 0, 0, 0, time.Local)
		t = t.AddDate(0, 0, -1)
		return t.Day()
	}

	// Функция для получения конкретного дня месяца.
	getTargetDay := func(year int, month time.Month, day int) (time.Time, error) {
		var targetDay int
		if day > 0 {
			lastDay := lastDayOfMonth(year, month)
			if day > lastDay {
				return time.Time{}, fmt.Errorf("день %d не существует в месяце %s %d", day, month, year)
			}
			targetDay = day
		} else if day == -1 {
			targetDay = lastDayOfMonth(year, month)
		} else if day == -2 {
			lastDay := lastDayOfMonth(year, month)
			if lastDay < 2 {
				return time.Time{}, fmt.Errorf("недопустимый предпоследний день для месяца %s %d", month, year)
			}
			targetDay = lastDay - 1
		} else {
			return time.Time{}, fmt.Errorf("недопустимый день месяца: %d", day)
		}

		candidateDate := time.Date(year, month, targetDay, 0, 0, 0, 0, time.Local)
		return candidateDate, nil
	}

	// Собираем список месяцев для поиска.
	searchMonths := []time.Month{}
	if len(months) == 0 {
		// Если месяцы не указаны, рассматриваем все месяцы.
		for m := time.January; m <= time.December; m++ {
			searchMonths = append(searchMonths, m)
		}
	} else {
		searchMonths = append(searchMonths, months...)
	}

	// Инициализируем переменную для хранения самой ранней даты.
	var earliestDate time.Time

	// Перебираем все дни и находим ближайшие даты.
	for _, day := range days {
		// Определяем начальный год и месяц для поиска.
		currentYear := now.Year()
		currentMonth := now.Month()

		if len(months) > 0 {
			// Месяцы указаны
			for _, m := range months {
				year := currentYear
				for {
					cDate, err := getTargetDay(year, m, day)
					if err != nil {
						break // Некорректный день для месяца
					}
					if cDate.After(now) && cDate.After(dateTime) {
						if earliestDate.IsZero() || cDate.Before(earliestDate) {
							earliestDate = cDate
						}
						break // Найдена подходящая дата для этого дня и месяца
					}
					year++
				}
			}
		} else {
			// Месяцы не указаны
			// Начинаем с текущего месяца и продвигаемся вперёд
			year := currentYear
			month := currentMonth
			for {
				cDate, err := getTargetDay(year, month, day)
				if err != nil {
					// Некорректный день для месяца, пропускаем
					month++
					if month > time.December {
						month = time.January
						year++
					}
					continue
				}
				if cDate.After(now) && cDate.After(dateTime) {
					if earliestDate.IsZero() || cDate.Before(earliestDate) {
						earliestDate = cDate
					}
					break // Найдена подходящая дата для этого дня
				}
				// Переходим к следующему месяцу
				month++
				if month > time.December {
					month = time.January
					year++
				}
			}
		}
	}

	if earliestDate.IsZero() {
		return "", fmt.Errorf("не удалось найти дату повторения")
	}

	return earliestDate.Format("20060102"), nil
}
