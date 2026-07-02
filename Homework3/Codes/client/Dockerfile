FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download

COPY . .
RUN go build -o /client .

FROM alpine:3.19

WORKDIR /app
COPY --from=builder /client /app/client

ENTRYPOINT ["/app/client"]
