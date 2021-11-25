FROM golang:1.17-alpine

WORKDIR /app

RUN apk add build-base

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY *.go ./

RUN go build -tags musl -o /pusher

CMD ["sh", "-c", "/pusher --enableRedis --sessionStoreAddr ${REDIS_ADDR} --kafkaTopic=${KAFKA_TOPIC} --kafkaBrokerServer=${KAFKA_BROKER_SERVERS}"]
