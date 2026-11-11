FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /apigateway-apiv1 ./examples/gcp/apigateway-apiv1
CMD ["/apigateway-apiv1"]
