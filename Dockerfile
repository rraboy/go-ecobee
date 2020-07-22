# from https://codefresh.io/docs/docs/learn-by-example/golang/golang-hello-world/
# ==== BUILDING ====
FROM golang:1-alpine AS build_base

RUN apk add --no-cache git
WORKDIR /tmp/build-dir

COPY . .

RUN go mod download
RUN CGO_ENABLED=0 go build -o /tmp/bin/go-ecobee .

# ==== RUN ====
FROM alpine:3

COPY --from=build_base /tmp/bin/go-ecobee /app/go-ecobee/go-ecobee

WORKDIR /app/go-ecobee
VOLUME "/app/go-ecobee/.go-ecobee.yaml"
VOLUME "/app/go-ecobee/.go-ecobee-authcache"
ENTRYPOINT ["/app/go-ecobee/go-ecobee"]
