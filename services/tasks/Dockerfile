# ── Stage 1: builder ──────────────────────────────────────────────────────────
 FROM golang:1.25-alpine AS builder

WORKDIR /app

# Копируем go.mod и go.sum из корня (кешируем зависимости отдельным слоем)
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь исходный код
COPY . .

# Собираем бинарник сервиса tasks
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/tasks ./services/tasks/cmd/tasks

# ── Stage 2: runner ───────────────────────────────────────────────────────────
FROM alpine:3.20

# Сертификаты нужны для HTTPS-вызовов изнутри контейнера
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Копируем только бинарник из builder
COPY --from=builder /bin/tasks /app/tasks

# Информационно: сервис слушает этот порт
EXPOSE 8082

# Запуск
ENTRYPOINT ["/app/tasks"]