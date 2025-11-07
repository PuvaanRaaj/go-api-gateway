FROM golang:1.22-alpine

WORKDIR /app
COPY . .

RUN go build -o service-a

EXPOSE 8081
CMD ["./service-a"]
