package main

import (
	"fmt"
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

	// Определяем путь к исполняемому файлу приложения
	for _, e := range os.Environ() {
		fmt.Println(e)
	}
	wd, _ := os.Getwd()
	fmt.Println("Current working directory:", wd)

	appPath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}

	// Устанавливаем путь к файлу базы данных
	dbFile := filepath.Join(filepath.Dir(appPath), "scheduler.db")
	if envDBFile := os.Getenv("TODO_DBFILE"); envDBFile != "" {
		// Если путь к базе данных задан в переменной окружения, используем его
		dbFile = envDBFile
	}

	// Проверяем, существует ли файл базы данных
	_, err = os.Stat(dbFile)
	var install bool
	if err != nil {
		// Если файл базы данных отсутствует, отмечаем, что нужно создать новую базу
		install = true
	}

	if install {
		// Создаём новую базу данных
		createDB(dbFile)
		log.Println("Создана новая база данных и таблица 'scheduler', путь:", dbFile)
	} else {
		// Открываем существующую базу данных
		openDB(dbFile)
		log.Println("База данных уже существует.")
	}
	// Закрываем базу данных при завершении программы
	defer db.Close()

	// Указываем путь к статическим файлам для веб-интерфейса
	webDir := "./web"
	port := "7540" // Устанавливаем порт по умолчанию
	if envPort := os.Getenv("TODO_PORT"); envPort != "" {
		// Если порт задан в переменной окружения, используем его
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
		// Если сервер не может запуститься, выводим ошибку и завершаем программу
		log.Fatal(err)
	}
}
