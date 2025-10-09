# Stage 1: Build React Frontend
FROM node:18-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go Backend
FROM golang:1.24 AS backend-builder
RUN apt-get update && apt-get upgrade -y && apt-get clean
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ingester ./cmd/ingester

# Stage 3: Final Image
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy backend binary
COPY --from=backend-builder /app/ingester .
COPY --from=backend-builder /app/configs ./configs

# Copy frontend build
COPY --from=frontend-builder /app/frontend/build ./static

EXPOSE 8080

CMD ["./ingester"]