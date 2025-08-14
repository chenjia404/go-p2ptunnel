FROM golang:1.25-alpine as builder

RUN apk add --no-cache gcc musl-dev linux-headers git

WORKDIR /build

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY . ./

RUN  go build  -ldflags="-w -s" -o /build/go-p2ptunnel .

FROM alpine:3

WORKDIR /
RUN apk update --no-cache && apk upgrade && apk add --no-cache ca-certificates

COPY --from=builder /build/go-p2ptunnel /app/go-p2ptunnel

ENTRYPOINT   ["/app/go-p2ptunnel"]
