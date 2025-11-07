FROM golang:1.22-alpine

WORKDIR /app
COPY . .

RUN go build -o gateway ./cmd/gateway

EXPOSE 8080
CMD ["./gateway"]
