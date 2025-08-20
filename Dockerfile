# Get Go image from DockerHub.
FROM golang:1.25 AS base

# Set working directory.
WORKDIR /go/src/omed

# Copy dependency locks so we can cache.
COPY go.mod go.sum ./

# Get all of our dependencies.
RUN go mod download

# Copy all of our remaining application.
COPY . .

EXPOSE 3000

FROM base AS dev

ENV OMED_APP_ENV "development"

# Set working directory.
RUN go install github.com/air-verse/air@latest

CMD ["air", "api.go", "-b", "0.0.0.0"]

FROM base AS build

# Build our application.
RUN CGO_ENABLED=0 GOOS=linux go build -o api ./api.go

# Use 'scratch' image for super-mini build.
FROM scratch AS prod

ENV OMED_APP_ENV "production"

# Set working directory for this stage.
WORKDIR /omed

RUN mkdir /config

# Copy our compiled executable from the last stage.
COPY --from=build /go/src/omed/api .
COPY --from=build /go/src/omed/config/config.json /config/

# Run application and expose port 8080.
CMD ["./api"]
