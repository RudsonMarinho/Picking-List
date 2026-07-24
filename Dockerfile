# syntax=docker/dockerfile:1
# Multi-stage — build Go 1.23 → runtime distroless (sem shell/curl).
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /app/invtech-api ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /app/invtech-api /app/invtech-api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/invtech-api"]
