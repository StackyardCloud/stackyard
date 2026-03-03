FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /location-basic ./examples/aws/location/location-basic
ENTRYPOINT ["/location-basic"]
