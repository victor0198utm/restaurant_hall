FROM golang:1.16-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./
COPY /appDataH1 ./appData
COPY /models ./models

RUN go build -o /hall_app

CMD [ "/hall_app" ]