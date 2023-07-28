FROM golang:latest

WORKDIR /app

COPY . .

RUN go build -o weather-app

CMD ["./weather-app"]
