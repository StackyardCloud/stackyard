FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /config ./examples/aws/config
ENTRYPOINT ["/config"]
