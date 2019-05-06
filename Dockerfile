FROM golang:alpine

RUN apk update && apk add --no-cache git g++

ENV DB_URL ""
WORKDIR /upnext-backend
COPY . .

RUN go build -mod=vendor -o upnext-backend

CMD ./upnext-backend
