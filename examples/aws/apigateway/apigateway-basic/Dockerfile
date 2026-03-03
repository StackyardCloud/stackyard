FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /apigateway-basic ./examples/aws/apigateway/apigateway-basic
ENTRYPOINT ["/apigateway-basic"]
