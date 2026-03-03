FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /appconfig ./examples/aws/appconfig/appconfig
ENTRYPOINT ["/appconfig"]
