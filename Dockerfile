FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

FROM alpine:3.22

WORKDIR /app

RUN addgroup -S atlas && adduser -S atlas -G atlas

COPY --from=builder /out/api /app/api
COPY migrations /app/migrations

USER atlas

EXPOSE 8080

ENTRYPOINT ["/app/api"]
