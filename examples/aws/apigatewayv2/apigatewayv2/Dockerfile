FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /apigatewayv2 ./examples/aws/apigatewayv2/apigatewayv2
ENTRYPOINT ["/apigatewayv2"]
