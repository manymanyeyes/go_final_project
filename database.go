package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// createDB создает новую базу данных и таблицу scheduler.
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
	} else {
		// Проверяем, является ли search датой
		parsedDate, dateErr := time.Parse("02.01.2006", search)
		if dateErr == nil {
			// Если это дата, выбираем задачи на эту дату
			rows, err = db.Query("SELECT id, date, title, comment, repeat FROM scheduler WHERE date = ? LIMIT ?", parsedDate.Format("20060102"), limit)
		} else {
			// Если это строка, ищем в title и comment
			likeSearch := "%" + search + "%"
			rows, err = db.Query("SELECT id, date, title, comment, repeat FROM scheduler WHERE title LIKE ? OR comment LIKE ? ORDER BY date LIMIT ?", likeSearch, likeSearch, limit)
		}
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		err := rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// addTask добавляет новую задачу в базу данных и возвращает ее ID.
func addTask(task Task) (int, error) {
	stmt, err := db.Prepare("INSERT INTO scheduler(date, title, comment, repeat) VALUES(?,?,?,?)")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	result, err := stmt.Exec(task.Date, task.Title, task.Comment, task.Repeat)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
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
		return err
	}

	// Проверяем, было ли обновлено хотя бы одно значение
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf(ErrorTaskNotFound)
	}

	return nil
}

// deleteTaskByID удаляет задачу из базы данных по идентификатору
func deleteTaskByID(id string) (bool, error) {
	result, err := db.Exec("DELETE FROM scheduler WHERE id = ?", id)
	if err != nil {
		return false, err // Ошибка запроса
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil // true, если что-то было удалено
}
