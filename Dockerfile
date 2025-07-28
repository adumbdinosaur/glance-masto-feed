FROM golang:1.24

WORKDIR /app
COPY . .

RUN go build -o app .

EXPOSE 9000

CMD ["./app"]