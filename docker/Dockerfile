FROM golang:1.26.0-alpine3.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /app/zink ./cmd/zink/main.go


FROM scratch

COPY --from=builder /app/zink /zink

EXPOSE 80

ENTRYPOINT ["/zink"]
