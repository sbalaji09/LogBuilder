# Build stage
FROM golang:1.24 AS builder

RUN apt-get update && apt-get upgrade -y && apt-get clean

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ingester ./cmd/ingester

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/ingester .
COPY --from=builder /app/configs ./configs

# Expose port
EXPOSE 8080

# Command to run
CMD ["./ingester"]