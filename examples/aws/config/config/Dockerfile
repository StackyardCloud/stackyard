FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /config ./examples/aws/config/config
ENTRYPOINT ["/config"]
