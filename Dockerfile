# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kpi-service .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/kpi-service .

# Create output directory
RUN mkdir -p /output

# Set default environment variables
ENV MONGO_URI=mongodb://localhost:27017
ENV MONGO_DB=trustroots
ENV NOSTR_RELAYS=wss://relay.trustroots.org,wss://relay.nomadwiki.org
ENV OUTPUT_PATH=/output/kpi.json
ENV UPDATE_INTERVAL_MINUTES=60

# Expose port (if needed for health checks)
EXPOSE 8080

# Run the service
CMD ["./kpi-service"]
