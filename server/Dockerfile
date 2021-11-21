# syntax=docker/dockerfile:1

# First step, build react app.
FROM node:12.18.1 as build-stage
ENV NODE_ENV=production

WORKDIR /app/client

COPY client/package*.json /app/client/
RUN npm install --production
COPY ./client /app/client/
RUN npm run build

# Second, build the server.
FROM golang:1.17-alpine

WORKDIR /app

COPY --from=build-stage /app/client/build/ /app/client/build

RUN apk add build-base

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY *.go ./

RUN go build -tags musl -o /server

EXPOSE 3000

CMD ["sh", "-c", "/server --enableDatabase --enableRedis  --databaseHost ${POSTGRES_HOST} --databasePort ${POSTGRES_PORT} --databaseUser ${POSTGRES_USER} --databasePassword ${POSTGRES_PASSWORD} --databaseName ${POSTGRES_DBNAME}"]
