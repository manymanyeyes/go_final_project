package main

// Task представляет задачу в планировщике
type Task struct {
	ID      string `json:"id,omitempty"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

// Credentials содержит данные для аутентификации
type Credentials struct {
	Password string `json:"password"` // Пароль, переданный в запросе.
}
