FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /routing-control-basic ./examples/routing-control/routing-control-basic
ENTRYPOINT ["/routing-control-basic"]
