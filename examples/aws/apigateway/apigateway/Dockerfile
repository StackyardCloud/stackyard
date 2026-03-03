FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /apigateway ./examples/aws/apigateway/apigateway
ENTRYPOINT ["/apigateway"]
