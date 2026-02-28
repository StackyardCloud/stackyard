FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /apigateway-advanced ./examples/apigateway/apigateway-advanced
ENTRYPOINT ["/apigateway-advanced"]
