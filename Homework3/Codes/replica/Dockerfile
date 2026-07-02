
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download

COPY . .
RUN go build -o /replica .


FROM alpine:3.19

WORKDIR /app
COPY --from=builder /replica /app/replica


ENTRYPOINT ["/app/replica"]
