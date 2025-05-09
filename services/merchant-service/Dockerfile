FROM golang:1.23-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Create gomicro directory
RUN mkdir -p /app/gomicro

# Copy gomicro module files
COPY gomicro/ /app/gomicro/

# Copy service-specific go.mod and go.sum files
COPY services/merchant-service/go.mod services/merchant-service/go.sum ./

# Fix replace directive in go.mod
RUN sed -i 's|replace github.com/suteetoe/gomicro => ../../gomicro|replace github.com/suteetoe/gomicro => /app/gomicro|g' go.mod

# Download dependencies
RUN go mod download

# Copy the service source code
COPY services/merchant-service/ .

# Fix replace directive in go.mod
RUN sed -i 's|replace github.com/suteetoe/gomicro => ../../gomicro|replace github.com/suteetoe/gomicro => /app/gomicro|g' go.mod

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