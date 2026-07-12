# Builder stage
FROM golang:1.26.3-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -gcflags="all=-N -l" -o main ./cmd/server

# Debig stage
FROM golang:1.26.3-alpine AS debug
WORKDIR /app
RUN go install github.com/go-delve/delve/cmd/dlv@latest
COPY --from=builder /app/main .
EXPOSE 8081 2345
CMD ["dlv", "--listen=:2345", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "./main"]

# Final stage
FROM alpine:latest AS production

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/main .

# Expose port (assuming 8080 or whatever is used, will be overridden by docker-compose if needed)
EXPOSE 8080

CMD ["./main"]
