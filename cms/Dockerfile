# Get Go image from DockerHub.
FROM golang:1.25 AS build

# Set working directory.
WORKDIR /compiler

# Copy dependency locks so we can cache.
COPY go.mod go.sum /compiler/

# Get all of our dependencies.
RUN go mod download

# Copy all of our remaining application.
COPY . .

# Build our application.
RUN CGO_ENABLED=0 GOOS=linux go build -o cms ./cmd/api/main.go

# Use 'scratch' image for super-mini build.
FROM scratch AS prod

# Set working directory for this stage.
WORKDIR /omed

# Copy our compiled executable from the last stage.
COPY --from=build /compiler/cms .

# Run application and expose port 8080.
EXPOSE 3000
CMD ["./cms"]
