FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /apigateway-advanced ./examples/aws/apigateway/apigateway-advanced
ENTRYPOINT ["/apigateway-advanced"]
