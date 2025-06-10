# Этап сборки
FROM golang:1.23.4-alpine AS builder

# Установка зависимостей для сборки
RUN apk add --no-cache git make

WORKDIR /app

# Копируем только файлы модулей для кэширования
COPY ./app/go.* ./
RUN go mod download

# Копируем все исходники
COPY ./app .

# Создаем директорию для результатов
RUN mkdir -p /app/results

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Этап запуска
FROM gcr.io/distroless/static-debian12

WORKDIR /app

# Копируем бинарник и ресурсы
COPY --from=builder /app/main .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static
COPY --from=builder /app/batfiles ./batfiles

# Копируем созданную директорию результатов
COPY --from=builder /app/results ./results

EXPOSE 8080

# Не нужно явно создавать директорию (уже скопирована)
# USER root и RUN mkdir удалены

CMD ["./main"]