FROM golang:1.26-alpine AS builder

WORKDIR /usr/src/app

# Copy dependency manifests first
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the project source code
COPY . .

# Build using standard module mode
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o=./minurl-api ./cmd/api

# --- Final Run Stage ---
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /

# Copy the compiled binary from the builder stage
COPY --from=builder /usr/src/app/minurl-api /minurl-api

# Copy migrations folder
COPY --from=builder /usr/src/app/migrations /migrations

# Expose the application port
EXPOSE 4000

CMD ["/minurl-api"]
