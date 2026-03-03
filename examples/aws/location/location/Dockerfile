FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /location ./examples/aws/location/location
ENTRYPOINT ["/location"]
