package main

import (
	"errors"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

// authMiddleware — промежуточный обработчик для проверки аутентификации.
// Используется для маршрутов, требующих авторизации.
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем токен из куки
		cookie, err := r.Cookie("token")
		if err != nil {
			http.Error(w, "Аутентификация обязательна", http.StatusUnauthorized)
			return
		}

		// Расшифровываем токен
		tokenStr := cookie.Value
		claims := &jwt.RegisteredClaims{}
		_, err = jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil // Используем jwtKey для проверки подписи
		})
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				http.Error(w, "Token expired", http.StatusUnauthorized)
			} else {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
			}
			return
		}

		// Проверяем хэш пароля в Subject
		if claims.Subject != generateHash(os.Getenv("TODO_PASSWORD")) {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Если токен валиден, передаём запрос следующему обработчику
		next(w, r)
	}
}
