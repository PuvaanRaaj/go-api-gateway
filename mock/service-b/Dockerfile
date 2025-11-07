FROM golang:1.22-alpine

WORKDIR /app
COPY . .

RUN go build -o service-b

EXPOSE 8082
CMD ["./service-b"]
