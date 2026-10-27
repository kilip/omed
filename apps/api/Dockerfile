FROM arigaio/atlas:latest AS atlas

FROM golang:1.27-alpine AS build
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download
COPY apps/api/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=atlas /atlas /atlas
COPY --from=build /out/api /api
COPY --from=build /src/migrations /migrations
COPY --from=build /src/atlas.hcl /atlas.hcl
ENV OMED_API_PORT=8001
EXPOSE 8001
USER nonroot:nonroot
ENTRYPOINT ["/api"]
