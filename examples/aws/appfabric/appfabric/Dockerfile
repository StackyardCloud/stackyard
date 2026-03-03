FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /appfabric ./examples/aws/appfabric/appfabric
ENTRYPOINT ["/appfabric"]
