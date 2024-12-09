# Первый этап: сборка приложения
FROM golang:1.22.1 AS builder

# Устанавливаем системные инструменты для CGO и зависимости для SQLite
RUN apt-get update && apt-get install -y gcc g++ libc6-dev libsqlite3-dev

# Включаем CGO и задаём параметры сборки для ARM64
ARG CGO_ENABLED=1
ARG GOOS=linux
ARG GOARCH=arm64

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем файлы зависимостей
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код проекта
COPY . .

# Сборка исполняемого файла
RUN go build -o server

# Второй этап: создание минимального образа
FROM alpine:latest

# Устанавливаем SQLite CLI для работы с базой данных
RUN apt-get update && apt-get install -y sqlite3

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем скомпилированный бинарник из первого этапа
COPY --from=builder /app/server .

# Копируем фронтенд файлы
COPY --from=builder /app/web ./web

# Устанавливаем переменные окружения для работы приложения
ENV TODO_PASSWORD=12345
ENV TODO_PORT=7540
ENV TODO_DBFILE=/app/scheduler.db

# Указываем порт, который будет использовать приложение
EXPOSE 7540

# Указываем команду для запуска приложения
ENTRYPOINT ["./server"]
