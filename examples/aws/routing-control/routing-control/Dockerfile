FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /routing-control ./examples/aws/routing-control/routing-control
ENTRYPOINT ["/routing-control"]
