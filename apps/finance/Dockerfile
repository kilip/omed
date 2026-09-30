# ---- Build stage ----
FROM golang:1.27-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /src

# Cache dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy source (embedded files are inside internal/)
COPY . .

# Build a statically linked binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/finance ./cmd/api

# ---- Runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/finance /finance

EXPOSE 9002

ENTRYPOINT ["/finance"]
