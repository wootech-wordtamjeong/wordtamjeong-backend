# 빌드용
FROM golang:1.21-bookworm AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# main.go가 cmd/server 아래에 있을 때
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /build/kkomantl-server ./cmd/server

# 실행용
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# 빌더에서 만든 바이너리만 복사
COPY --from=builder /build/kkomantl-server /app/kkomantl-server
COPY --from=builder /build/data ./data

EXPOSE 8080

ENTRYPOINT ["/app/kkomantl-server"]
