FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /appconfig ./examples/aws/appconfig
ENTRYPOINT ["/appconfig"]
