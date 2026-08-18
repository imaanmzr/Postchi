# Build stage
FROM golang:1.26.3-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY backend/go.mod backend/go.sum* ./
RUN go mod download
COPY backend/ ./
COPY migrations /migrations
RUN CGO_ENABLED=0 go build -o /postchi-api ./cmd/api \
 && CGO_ENABLED=0 go build -o /postchi-migrate ./cmd/migrate

# Runtime stage
FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /postchi-api /app/postchi-api
COPY --from=builder /postchi-migrate /app/postchi-migrate
COPY --from=builder /migrations /migrations
EXPOSE 8080
ENV MIGRATIONS_PATH=file:///migrations
ENV AUTO_MIGRATE=true
ENV DB_READY_TIMEOUT=60s
CMD ["/app/postchi-api"]
