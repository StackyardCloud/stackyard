FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /appconfig ./examples/aws/appconfig
ENTRYPOINT ["/appconfig"]
