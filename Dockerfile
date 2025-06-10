# Этап сборки
FROM golang:1.23.4 AS builder
WORKDIR /app
COPY ./app ./

# Скачиваем зависимости (если есть go.mod)
RUN go mod download

# Собираем бинарник
RUN go build -o main main.go handlers.go database.go batfiles.go hosts.go models.go

# Этап запуска
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static
COPY --from=builder /app/batfiles ./batfiles

# Создаем папку для результатов
RUN mkdir results

EXPOSE 8080
CMD ["./main"]