package main

import "errors"

// Константы ошибок
var (
	ErrorMethodNotAllowed  = errors.New("метод не поддерживается")
	ErrorInvalidID         = errors.New("не указан идентификатор задачи")
	ErrorTaskNotFound      = errors.New("задача не найдена")
	ErrorDecodingJSON      = errors.New("ошибка декодирования JSON")
	ErrorTaskTitleRequired = errors.New("не указан заголовок задачи")
	ErrorRepeatRule        = errors.New("некорректное правило повторения")
	ErrorDeletingTask      = errors.New("ошибка при удалении задачи")
	ErrorUpdatingTask      = errors.New("ошибка при обновлении задачи")
	ErrorAddingTask        = errors.New("ошибка при добавлении задачи")
)

// jwtKey — секретный ключ для подписи JWT-токенов.
var jwtKey = []byte("go_final_project")
