# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install certificates and git
RUN apk add --no-cache ca-certificates git

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build statically linked binary without debug symbols
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/zitadel-actions-api ./cmd/server

# Runtime stage: Minimal distroless container for security and minimal footprint
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /

# Copy binary from builder
COPY --from=builder /app/zitadel-actions-api /zitadel-actions-api

# Use nonroot UID 65532 from distroless
USER 65532:65532

EXPOSE 8080

ENTRYPOINT ["/zitadel-actions-api"]
