FROM golang:1.23.3-bookworm
COPY . /go/src/github.com/flowerinthenight/freyr/
WORKDIR /go/src/github.com/flowerinthenight/freyr/
RUN GOOS=linux go build -v -trimpath -o freyr .

FROM debian:stable-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y curl ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app/
COPY --from=0 /go/src/github.com/flowerinthenight/freyr/freyr .
ENTRYPOINT ["/app/freyr"]
CMD ["run", "--logtostderr", "--db=projects/v/instances/v/databases/v", "--lock-table=v", "--log-table=v"]
