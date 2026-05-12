FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o bin/api ./cmd/api

FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/bin/api ./bin/api
COPY --from=builder /app/migrations ./migrations
EXPOSE 3001
CMD ["./bin/api"]
