# Build frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Build backend
# Using golang:latest to avoid version availability issues
FROM golang:latest AS backend-builder
WORKDIR /app
RUN apt-get update && apt-get install -y git ca-certificates && rm -rf /var/lib/apt/lists/*
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o server ./cmd/server

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -S appgroup && \
    adduser -S appuser -G appgroup
WORKDIR /app
COPY --from=backend-builder --chown=appuser:appgroup /app/server .
COPY --from=frontend-builder --chown=appuser:appgroup /app/frontend/dist ./frontend/dist
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1
CMD ["./server"]