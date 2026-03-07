FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /routing-control ./examples/aws/routing-control
ENTRYPOINT ["/routing-control"]
