FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /appfabric-advanced ./examples/appfabric/appfabric-advanced
ENTRYPOINT ["/appfabric-advanced"]
