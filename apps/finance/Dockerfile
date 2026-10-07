FROM golang:1.27-alpine AS build
WORKDIR /src
COPY apps/finance/go.mod apps/finance/go.sum ./
RUN go mod download
COPY apps/finance/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
ENV FIN_PORT=8002
EXPOSE 8002
USER nonroot:nonroot
ENTRYPOINT ["/api"]

