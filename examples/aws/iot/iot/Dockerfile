FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot ./examples/aws/iot/iot
ENTRYPOINT ["/iot"]
