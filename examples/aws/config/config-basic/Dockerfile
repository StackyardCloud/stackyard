FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /config-basic ./examples/aws/config/config-basic
ENTRYPOINT ["/config-basic"]
