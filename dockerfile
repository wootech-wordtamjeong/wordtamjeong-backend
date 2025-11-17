# 빌드용
FROM golang:1.21-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ***-server ./cmd/server

# 실행용
FROM gcr.io/distroless/base-debian12

WORKDIR /app

COPY --from=builder /app/kkomantl-server .

EXPOSE 8080

ENTRYPOINT ["/app/kkomantl-server"]
