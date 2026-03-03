FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /routing-control-advanced ./examples/aws/routing-control/routing-control-advanced
ENTRYPOINT ["/routing-control-advanced"]
