FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /apigatewayv2-basic ./examples/aws/apigatewayv2/apigatewayv2-basic
ENTRYPOINT ["/apigatewayv2-basic"]
