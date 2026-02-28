FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /apigateway-basic ./examples/apigateway/apigateway-basic
ENTRYPOINT ["/apigateway-basic"]
