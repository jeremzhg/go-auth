FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /auth-service ./cmd/auth-service

FROM alpine:latest

COPY --from=builder /auth-service /auth-service
COPY --from=builder /app/migrations /migrations
COPY --from=builder /app/model.conf /model.conf

EXPOSE 8080

CMD ["/auth-service"]
