FROM golang:1.17-alpine

WORKDIR /app

RUN apk add build-base

COPY go.mod ./
COPY go.sum ./
COPY client/ ./

RUN go mod download

COPY *.go ./

RUN go build -tags musl -o /docker-goserver

EXPOSE 3000

CMD ["sh", "-c", "/docker-goserver --enableKafka $GOSERVER_POSTGRES_HOST $GOSERVER_POSTGRES_PORT $GOSERVER_POSTGRES_USER $GOSERVER_POSTGRES_PASSWORD $GOSERVER_POSTGRES_DBNAME"]
