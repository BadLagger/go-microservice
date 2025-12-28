FROM golang:1.24-alpine AS builder

WORKDIR /app

# Копируем модули и скачиваем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь код и собираем
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Финальный образ - используем scratch вместо alpine
FROM scratch

# Копируем SSL сертификаты из builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 9898
CMD ["./main"]