package main

import (
	"log"
	"net/http"
	"os"
)

// registerRoute регистрирует маршруты с учётом необходимости аутентификации
func registerRoute(path string, handler http.HandlerFunc, useAuth bool) {
	if useAuth {
		http.HandleFunc(path, authMiddleware(handler))
	} else {
		http.HandleFunc(path, handler)
	}
}

func main() {
	// Проверяем наличие пароля в переменной окружения
	password := os.Getenv("TODO_PASSWORD")
	useAuth := password != "" // Определяем, нужна ли аутентификация

	// Инициализируем базу данных
	if err := InitDB(); err != nil {
		log.Fatalf("Ошибка инициализации базы данных: %v", err)
	}
	defer db.Close()

	// Указываем путь к статическим файлам для веб-интерфейса
	webDir := "./web"
	port := "7540" // Устанавливаем порт по умолчанию
	if envPort := os.Getenv("TODO_PORT"); envPort != "" {
		port = envPort
	}

	// Настраиваем маршруты
	// web-файлы
	http.Handle("/", http.FileServer(http.Dir(webDir)))
	// GET-запрос для вычисления следующей даты
	registerRoute("/api/nextdate", nextDateHandler, false)
	// Эндпоинт для аутентификации, регистрируется только если нужен пароль
	if useAuth {
		log.Println("Аутентификация включена")
		http.HandleFunc("/api/signin", signHandler)
	} else {
		log.Println("Аутентификация отключена")
	}
	// Регистрация маршрутов, зависящих от аутентификации
	registerRoute("/api/task", taskHandler, useAuth)          // Объединённый обработчик для /api/task
	registerRoute("/api/tasks", getTasksHandler, useAuth)     // GET-запрос для списка задач
	registerRoute("/api/task/done", taskDoneHandler, useAuth) // POST-запрос для выполнения задачи

	// Запускаем HTTP-сервер
	log.Printf("Сервер запущен на порту %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
