# Build stage
FROM golang:1.21-bookworm AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /build/kkomantl-server ./cmd/server

# Runtime stage
FROM gcr.io/distroless/base-debian12

WORKDIR /app

COPY --from=builder /build/kkomantl-server /app/kkomantl-server
COPY --from=builder /build/data ./data

EXPOSE 8080

ENTRYPOINT ["/app/kkomantl-server"]
