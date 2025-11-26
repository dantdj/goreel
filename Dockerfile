# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first to properly cache dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Use CGO_ENABLED=0 to create a statically linked binary
# This helps prevent subtle versioning issues between musl versions
RUN CGO_ENABLED=0 GOOS=linux go build -o goreel .

# Run stage
FROM alpine:latest

# ffmpeg is required for video processing
# ca-certificates is required for HTTPS requests
RUN apk add --no-cache ffmpeg ca-certificates

WORKDIR /root/

COPY --from=builder /app/goreel .

EXPOSE 8089

CMD ["./goreel"]
