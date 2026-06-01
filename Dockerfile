FROM golang:1.25 AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o /out/app ./cmd/spark-whatsapp-module

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /out/app /app/app

RUN mkdir -p /app/data

ENV HTTP_ADDRESS=:8080
ENV SQLITE_PATH=/app/data/spark-whatsapp-module.db

EXPOSE 8080

CMD ["/app/app"]
