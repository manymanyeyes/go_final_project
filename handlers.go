package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"os"
	"time"
)

// signHandler — обработчик для аутентификации пользователей.
// Проверяет пароль, сохраняет токен в куку и возвращает его клиенту.
func signHandler(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Проверяем пароль
	expectedPassword := os.Getenv("TODO_PASSWORD")
	if creds.Password != expectedPassword {
		http.Error(w, `{"error":"Неверный пароль"}`, http.StatusUnauthorized)
		return
	}

	// Генерируем JWT-токен
	expirationTime := time.Now().Add(8 * time.Hour)
	claims := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		Issuer:    "todo-planner",
		Subject:   generateHash(expectedPassword), // Хэш пароля записывается в Subject
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		http.Error(w, `{"error":"Ошибка создания токена"}`, http.StatusInternalServerError)
		return
	}

	// Сохраняем токен в куку
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenString,
		Expires:  expirationTime,
		HttpOnly: true,
	})

	// Возвращаем токен клиенту
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
}

// generateHash генерирует SHA-256 хэш для строки.
func generateHash(password string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(password)))
}

// taskHandler обрабатывает запросы к /api/task
// В зависимости от метода запроса (GET, POST, PUT) вызывает соответствующие функции
func taskHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Обработка GET-запроса: получение задачи по идентификатору
		getTaskHandler(w, r)
	case http.MethodPost:
		// Обработка POST-запроса: добавление новой задачи
		addTaskHandler(w, r)
	case http.MethodPut:
		// Обработка PUT-запроса: редактирование существующей задачи
		editTaskHandler(w, r)
	case http.MethodDelete:
		// Обработка DELETE-запроса: удаление задачи
		deleteTaskHandler(w, r)
	default:
		// Обработка неподдерживаемых методов
		http.Error(w, ErrorMethodNotAllowed, http.StatusMethodNotAllowed)
	}
}

// Обработчик для вычисления следующей даты
func nextDateHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем параметры запроса
	nowStr := r.FormValue("now")
	dateStr := r.FormValue("date")
	repeat := r.FormValue("repeat")

	// Преобразуем 'now' в time.Time
	now, err := time.Parse("20060102", nowStr)
	if err != nil {
		http.Error(w, "Неверный формат даты 'now'", http.StatusBadRequest)
		return
	}

	// Вызываем функцию NextDate для вычисления следующей даты
	nextDate, err := NextDate(now, dateStr, repeat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Возвращаем результат в формате 20060102
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	w.Write([]byte(nextDate))
}

// Обработчик для получения задачи по идентификатору
func getTaskHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем параметр id
	id := r.FormValue("id")
	if id == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "Не указан идентификатор"})
		return
	}

	// Получаем задачу из базы данных
	task, err := getTaskByID(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": ErrorTaskNotFound})
		return
	}

	// Возвращаем задачу в формате JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// Обработчик для получения списка ближайших задач
func getTasksHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем параметры запроса
	search := r.FormValue("search")
	limit := 50 // Максимальное количество записей

	// Получаем задачи из базы данных
	tasks, err := getTasks(search, limit)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка при получении списка задач"})
		return
	}

	// Если задач нет, возвращаем пустой список
	if tasks == nil {
		tasks = []Task{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tasks": tasks})
}

// Обработчик для добавления новой задачи
func addTaskHandler(w http.ResponseWriter, r *http.Request) {
	// Проверяем, что запрос выполнен с методом POST
	if r.Method != http.MethodPost {
		http.Error(w, ErrorMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var task Task
	// Декодируем тело запроса в структуру Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": ErrorDecodingJSON})
		return
	}

	// Проверяем обязательные поля
	if task.Title == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": ErrorTaskTitleRequired})
		return
	}

	// Проверяем и обрабатываем данные задачи
	now := time.Now()
	if err := checkTaskData(&task, now); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Добавляем задачу в базу данных
	id, err := addTask(task)
	if err != nil {
		http.Error(w, "Ошибка при добавлении задачи", http.StatusInternalServerError)
		return
	}

	// Возвращаем успешный ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

// Обработчик для редактирования задачи
func editTaskHandler(w http.ResponseWriter, r *http.Request) {
	// Проверяем, что запрос выполнен с методом PUT
	if r.Method != http.MethodPut {
		http.Error(w, ErrorMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var task Task
	// Декодируем тело запроса в структуру Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": ErrorDecodingJSON})
		return
	}

	// Проверяем обязательные поля
	if task.ID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": ErrorInvalidID})
		return
	}
	if task.Title == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": ErrorTaskTitleRequired})
		return
	}

	// Проверяем и обрабатываем данные задачи
	now := time.Now()
	if err := checkTaskData(&task, now); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Обновляем задачу в базе данных
	err = updateTask(task)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": ErrorTaskNotFound})
		return
	}

	// Возвращаем успешный ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{})
}

// deleteTaskHandler обрабатывает запрос на удаление задачи
func deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	// Устанавливаем заголовок для ответа в формате JSON
	w.Header().Set("Content-Type", "application/json")

	// Получаем идентификатор задачи из параметров запроса
	id := r.FormValue("id")
	if id == "" {
		// Если идентификатор отсутствует, возвращаем ошибку
		json.NewEncoder(w).Encode(map[string]string{"error": ErrorInvalidID})
		return
	}

	// Вызываем функцию удаления задачи по идентификатору
	deleted, err := deleteTaskByID(id)
	if err != nil {
		// Если произошла ошибка выполнения запроса к базе данных, возвращаем ошибку
		json.NewEncoder(w).Encode(map[string]string{"error": ErrorDeletingTask})
		return
	}

	// Если задача не была удалена
	if !deleted {
		json.NewEncoder(w).Encode(map[string]string{"error": ErrorTaskNotFound})
		return
	}

	// Если всё прошло успешно, возвращаем пустой JSON-ответ
	json.NewEncoder(w).Encode(map[string]interface{}{})
}

// taskDoneHandler обрабатывает запрос для завершения задачи
func taskDoneHandler(w http.ResponseWriter, r *http.Request) {
	// Устанавливаем заголовок для ответа в формате JSON
	w.Header().Set("Content-Type", "application/json")

	// Получаем идентификатор задачи из параметров запроса
	id := r.FormValue("id")
	if id == "" {
		// Если идентификатор отсутствует, возвращаем ошибку
		json.NewEncoder(w).Encode(map[string]string{"error": ErrorInvalidID})
		return
	}

	// Получаем задачу по идентификатору
	task, err := getTaskByID(id)
	if err != nil {
		// Если задача не найдена, возвращаем ошибку
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Если задача одноразовая (repeat пустой), удаляем её
	if task.Repeat == "" {
		deleted, err := deleteTaskByID(id)
		if err != nil {
			// Если возникла ошибка при удалении задачи
			json.NewEncoder(w).Encode(map[string]string{"error": ErrorDeletingTask})
			return
		}

		if !deleted {
			// Если задача не была удалена
			json.NewEncoder(w).Encode(map[string]string{"error": ErrorTaskNotFound})
			return
		}

		// Возвращаем успешный ответ для одноразовой задачи
		json.NewEncoder(w).Encode(map[string]interface{}{})
		return
	}

	// Если задача повторяющаяся, вычисляем следующую дату
	now := time.Now()
	nextDate, err := NextDate(now, task.Date, task.Repeat)
	if err != nil {
		// Если ошибка в вычислении следующей даты
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка вычисления следующей даты"})
		return
	}

	// Обновляем задачу с новой датой
	_, err = db.Exec("UPDATE scheduler SET date = ? WHERE id = ?", nextDate, id)
	if err != nil {
		// Если произошла ошибка обновления задачи
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка при обновлении задачи"})
		return
	}

	// Возвращаем успешный ответ для повторяющейся задачи
	json.NewEncoder(w).Encode(map[string]interface{}{})
}
