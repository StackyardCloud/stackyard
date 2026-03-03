FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /appfabric-basic ./examples/aws/appfabric/appfabric-basic
ENTRYPOINT ["/appfabric-basic"]
