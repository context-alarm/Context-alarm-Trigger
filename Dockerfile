# Build stage
FROM golang:1.21 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/alarm-checker \
    main.go config.go database_helper.go gemini_rest.go scheduler.go

# Runtime stage
FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=build /app/alarm-checker /app/alarm-checker
USER nonroot:nonroot
ENTRYPOINT ["/app/alarm-checker"]