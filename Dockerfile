# Stage 1: Build React Frontend
FROM node:18-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci --legacy-peer-deps
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go Backend
FROM golang:1.24 AS backend-builder
RUN apt-get update && apt-get upgrade -y && apt-get clean
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ingester ./log_analytics_engine/cmd/ingester

# Stage 3: Final Image
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy backend binary
COPY --from=backend-builder /app/ingester .

# Copy frontend build to static directory (where your Go app expects it)
COPY --from=frontend-builder /app/frontend/build ./static

# Expose port
EXPOSE 8080

# Command to run
CMD ["./ingester"]
