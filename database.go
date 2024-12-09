package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB // Глобальная переменная для базы данных

// InitDB инициализирует глобальную переменную db.
// Если файл базы данных отсутствует, создаёт новую.
func InitDB() error {
	dbFile := os.Getenv("TODO_DBFILE")
	if dbFile == "" {
		dbFile = "scheduler.db"
	}

	if _, err := os.Stat(dbFile); err != nil {
		if os.IsNotExist(err) {
			db, err = createDB(dbFile)
			if err != nil {
				return fmt.Errorf("ошибка создания базы данных: %w", err)
			}
		} else {
			return fmt.Errorf("ошибка проверки базы данных: %w", err)
		}
	} else {
		db, err = openDB(dbFile)
		if err != nil {
			return fmt.Errorf("ошибка открытия базы данных: %w", err)
		}
	}

	return nil
}

// createDB создает новую базу данных и таблицу scheduler с индексом по полю date.
func createDB(dbFile string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}

	// Создание таблицы scheduler
	sqlStmt := `
    CREATE TABLE scheduler (
        id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        date TEXT,
        title TEXT,
        comment TEXT,
        repeat TEXT
    );
    `
	_, err = db.Exec(sqlStmt)
	if err != nil {
		db.Close() // Закрываем базу, если произошла ошибка
		return nil, fmt.Errorf("ошибка создания таблицы: %w", err)
	}

	// Создание индекса по полю date
	indexStmt := `
    CREATE INDEX idx_scheduler_date ON scheduler(date);
    `
	_, err = db.Exec(indexStmt)
	if err != nil {
		db.Close() // Закрываем базу, если произошла ошибка
		return nil, fmt.Errorf("ошибка создания индекса: %w", err)
	}

	return db, nil
}

// openDB открывает существующую базу данных.
func openDB(dbFile string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// getTaskByID возвращает задачу по идентификатору
func getTaskByID(id string) (Task, error) {
	var task Task
	err := db.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", id).
		Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		return Task{}, err
	}
	return task, nil
}

// getTasks получает задачи из базы данных
func getTasks(search string, limit int) ([]Task, error) {
	var rows *sql.Rows
	var err error

	if search == "" {
		// Если параметр search отсутствует, возвращаем ближайшие задачи
		rows, err = db.Query("SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date LIMIT ?", limit)
		if err != nil {
			return nil, fmt.Errorf("ошибка выполнения запроса для получения задач: %w", err)
		}
	} else {
		// Проверяем, является ли search датой
		parsedDate, dateErr := time.Parse("02.01.2006", search)
		if dateErr == nil {
			// Если это дата, выбираем задачи на эту дату
			rows, err = db.Query("SELECT id, date, title, comment, repeat FROM scheduler WHERE date = ? LIMIT ?", parsedDate.Format("20060102"), limit)
			if err != nil {
				return nil, fmt.Errorf("ошибка выполнения запроса для поиска задач по дате '%s': %w", search, err)
			}
		} else {
			// Если это строка, ищем в title и comment
			likeSearch := "%" + search + "%"
			rows, err = db.Query("SELECT id, date, title, comment, repeat FROM scheduler WHERE title LIKE ? OR comment LIKE ? ORDER BY date LIMIT ?", likeSearch, likeSearch, limit)
			if err != nil {
				return nil, fmt.Errorf("ошибка выполнения запроса для поиска задач по строковому запросу '%s': %w", search, err)
			}
		}
	}

	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		err := rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
		if err != nil {
			return nil, fmt.Errorf("ошибка при сканировании строки результата: %w", err)
		}
		tasks = append(tasks, task)
	}

	// Проверяем на ошибки после завершения итерации
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при обработке строк результата: %w", err)
	}

	return tasks, nil
}

// addTask добавляет новую задачу в базу данных и возвращает ее ID.
func addTask(task Task) (int, error) {
	stmt, err := db.Prepare("INSERT INTO scheduler(date, title, comment, repeat) VALUES(?,?,?,?)")
	if err != nil {
		return 0, fmt.Errorf("ошибка подготовки запроса для добавления задачи: %w", err)
	}
	defer stmt.Close()

	result, err := stmt.Exec(task.Date, task.Title, task.Comment, task.Repeat)
	if err != nil {
		return 0, fmt.Errorf("ошибка выполнения запроса для добавления задачи: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("ошибка получения ID вставленной задачи: %w", err)
	}

	return int(id), nil
}

// updateTask обновляет задачу в базе данных
func updateTask(task Task) error {
	result, err := db.Exec(
		"UPDATE scheduler SET date = ?, title = ?, comment = ?, repeat = ? WHERE id = ?",
		task.Date, task.Title, task.Comment, task.Repeat, task.ID,
	)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса на обновление задачи: %w", err)
	}

	// Проверяем, было ли обновлено хотя бы одно значение
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества обновленных строк: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("задача с ID %d не найдена для обновления: %w", task.ID, ErrorTaskNotFound)
	}

	return nil
}

// deleteTaskByID удаляет задачу из базы данных по идентификатору
func deleteTaskByID(id string) (bool, error) {
	result, err := db.Exec("DELETE FROM scheduler WHERE id = ?", id)
	if err != nil {
		return false, fmt.Errorf("ошибка выполнения запроса на удаление задачи с ID %s: %w", id, err) // Ошибка запроса
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("ошибка получения количества удаленных строк: %w", err)
	}

	return rowsAffected > 0, nil // true, если что-то было удалено
}
