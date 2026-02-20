# Build stage
FROM golang:1.26 AS builder

WORKDIR /app

# Copy go.mod and download dependencies
COPY go.mod ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o icebreaker-exporter ./cmd/icebreaker-exporter

# Run stage
FROM gcr.io/distroless/static-debian12:latest

# Set working directory for the runtime container
WORKDIR /

# Copy the compiled binary from the builder stage
COPY --from=builder /app/icebreaker-exporter /icebreaker-exporter

# Expose the default metrics port
EXPOSE 9877

# Set the command to run the exporter
ENTRYPOINT ["/icebreaker-exporter"]
