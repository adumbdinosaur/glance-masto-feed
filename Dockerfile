FROM golang:1.21-alpine

WORKDIR /app
COPY . .

RUN go mod init mastofeed && go build -o app .

EXPOSE 9000

CMD ["./app"]