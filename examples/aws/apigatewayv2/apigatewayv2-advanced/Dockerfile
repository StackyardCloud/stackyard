FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /apigatewayv2-advanced ./examples/aws/apigatewayv2/apigatewayv2-advanced
ENTRYPOINT ["/apigatewayv2-advanced"]
