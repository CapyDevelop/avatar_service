FROM golang:1.20-alpine as builder

RUN apk update && apk add --no-cache make

WORKDIR /usr/src/service
COPY go.mod .
COPY go.sum .

RUN go mod download
COPY . .

RUN go build -o build/main cmd/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /usr/src/service/build/main .
#COPY --from=builder /usr/src/service/config/config.yaml /app/config/config.yaml
CMD ["/app/main"]