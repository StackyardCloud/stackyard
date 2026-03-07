FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /config ./examples/aws/config
ENTRYPOINT ["/config"]
