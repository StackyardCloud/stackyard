FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /apigeeconnect-apiv1 ./examples/gcp/apigeeconnect-apiv1
CMD ["/apigeeconnect-apiv1"]
