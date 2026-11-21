FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /apigatewaymanagementapi ./examples/aws/apigatewaymanagementapi
ENTRYPOINT ["/apigatewaymanagementapi"]
