FROM golang:1.17-alpine

WORKDIR /app

RUN apk add build-base

COPY go.mod ./
COPY go.sum ./
COPY index.html ./

RUN go mod download

COPY *.go ./

RUN go build -tags musl -o /server

EXPOSE 3000

CMD ["sh", "-c", "/server --enableDatabase --enableRedis  --databaseHost ${POSTGRES_HOST} --databasePort ${POSTGRES_PORT} --databaseUser ${POSTGRES_USER} --databasePassword ${POSTGRES_PASSWORD} --databaseName ${POSTGRES_DBNAME}"]
