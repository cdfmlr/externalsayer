# Dockerfile for GitHub Actions CI

FROM golang:1.19.7-alpine3.16 AS builder

WORKDIR /app

# dependencies

COPY go.mod go.sum ./

RUN go mod download

# build

COPY . .

RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o externalsayer.out .

# scratch 似乎每次都会遇到问题，还是 alpine 吧。
FROM alpine:3.16 AS runner

COPY --chown=0:0 --from=builder /app/externalsayer.out /app/externalsayer.out

# VOLUME ["/app/config"]

CMD ["/app/externalsayer.out", "-c", "/app/config/config.yaml"]
