FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot ./examples/aws/iot
ENTRYPOINT ["/iot"]
