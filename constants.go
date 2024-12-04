package main

// Константы для маршрутов
const (
	RouteNextDate = "/api/nextdate"
	RouteTask     = "/api/task"
	RouteTasks    = "/api/tasks"
	RouteTaskDone = "/api/task/done"
	RouteSignIn   = "/api/signin"
)

// Константы ошибок
const (
	ErrorMethodNotAllowed  = "Метод не поддерживается"
	ErrorInvalidID         = "Не указан идентификатор задачи"
	ErrorTaskNotFound      = "Задача не найдена"
	ErrorDecodingJSON      = "Ошибка декодирования JSON"
	ErrorTaskTitleRequired = "Не указан заголовок задачи"
	ErrorRepeatRule        = "Некорректное правило повторения"
	ErrorDeletingTask      = "Ошибка при удалении задачи"
)

// jwtKey — секретный ключ для подписи JWT-токенов.
var jwtKey = []byte("go_final_project")
