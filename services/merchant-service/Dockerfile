FROM golang:1.23-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o merchant-service ./cmd/main.go

# Use a minimal alpine image for the final stage
FROM alpine:3.18

# Install CA certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/merchant-service .

# Expose the application port
EXPOSE 8080

# Set environment variables
ENV APP_ENV=production
ENV SERVER_PORT=8080

# Run the application
CMD ["./merchant-service"]