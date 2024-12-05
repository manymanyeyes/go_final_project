package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	// Проверяем наличие пароля в переменной окружения
	password := os.Getenv("TODO_PASSWORD")
	if password == "" {
		log.Fatal("TODO_PASSWORD не задан. Установите пароль в переменной окружения.")
	}

	// Получаем текущую рабочую директорию
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("Ошибка получения текущей директории:", err)
	}

	// Всегда создаём базу данных в корне проекта
	dbFile := filepath.Join(wd, "scheduler.db")
	if envDBFile := os.Getenv("TODO_DBFILE"); envDBFile != "" {
		dbFile = envDBFile
	}

	// Проверяем, существует ли файл базы данных
	_, err = os.Stat(dbFile)
	var install bool
	if err != nil {
		install = true // База данных отсутствует, её нужно создать
	}

	// Создаём или открываем базу данных
	if install {
		db, err = createDB(dbFile)
		if err != nil {
			log.Fatalf("Ошибка создания базы данных: %v", err)
		}
		log.Println("Создана новая база данных и таблица 'scheduler', путь:", dbFile)
	} else {
		db, err = openDB(dbFile)
		if err != nil {
			log.Fatalf("Ошибка открытия базы данных: %v", err)
		}
		log.Println("База данных уже существует.")
	}
	defer db.Close()

	// Указываем путь к статическим файлам для веб-интерфейса
	webDir := "./web"
	port := "7540" // Устанавливаем порт по умолчанию
	if envPort := os.Getenv("TODO_PORT"); envPort != "" {
		port = envPort
	}

	// Настраиваем маршруты
	http.Handle("/", http.FileServer(http.Dir(webDir)))             // web-файлы
	http.HandleFunc(RouteSignIn, signHandler)                       // Эндпоинт для аутентификации
	http.HandleFunc(RouteNextDate, nextDateHandler)                 // GET-запрос для вычисления следующей даты
	http.HandleFunc(RouteTask, authMiddleware(taskHandler))         // Объединённый обработчик для /api/task
	http.HandleFunc(RouteTasks, authMiddleware(getTasksHandler))    // GET-запрос для списка задач
	http.HandleFunc(RouteTaskDone, authMiddleware(taskDoneHandler)) // POST-запрос для выполнения задачи

	// Запускаем HTTP-сервер
	log.Printf("Сервер запущен на порту %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
