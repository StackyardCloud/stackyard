FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /location-advanced ./examples/aws/location/location-advanced
ENTRYPOINT ["/location-advanced"]
