# syntax=docker/dockerfile:1

# --- СТАДИЯ СБОРКИ ---
FROM golang:1.23 AS builder
WORKDIR /app

# Устанавливаем librdkafka-dev
RUN apt-get update && apt-get install -y librdkafka-dev

# Копируем файлы зависимостей и загружаем модули
COPY go.mod .
COPY go.sum .
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение с включенным CGO
RUN CGO_ENABLED=1 GOOS=linux go build -o app ./cmd/UserService/

# --- Финальный образ ---
FROM ubuntu:22.04
WORKDIR /app

# Копируем скомпилированное приложение
COPY --from=builder /app/app .
COPY ./config/local.yaml /app/config.yaml

# Объявляем порт (если требуется)
EXPOSE 8080

# Запускаем приложение
CMD ["./app"]
