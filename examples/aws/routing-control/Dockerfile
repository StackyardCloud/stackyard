FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /routing-control ./examples/aws/routing-control
ENTRYPOINT ["/routing-control"]
