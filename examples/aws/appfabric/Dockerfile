FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /appfabric ./examples/aws/appfabric
ENTRYPOINT ["/appfabric"]
