# Build Stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git for fetch dependencies
RUN apk add --no-cache git

# Copy go.mod and go.sum files
COPY go.mod go.sum* ./

# Download dependencies
# Since go.sum might be missing initially or out of sync due to manual editing
RUN go mod tidy


# Copy source code
COPY . .

# Build the main application
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server

# Build the seeding tool
RUN CGO_ENABLED=0 GOOS=linux go build -o seed ./cmd/seed

# Build the admin CLI tool
RUN CGO_ENABLED=0 GOOS=linux go build -o admin ./cmd/admin

# Run Stage
FROM alpine:latest

WORKDIR /app

# wget needed for Docker health checks; ca-certificates for HTTPS outbound calls; tzdata for timezone parsing
RUN apk add --no-cache wget ca-certificates tzdata

# Copy binaries from builder
COPY --from=builder /app/main .
COPY --from=builder /app/seed .
COPY --from=builder /app/admin .

# Expose port
EXPOSE 8081

# Command to run
CMD ["./main"]
