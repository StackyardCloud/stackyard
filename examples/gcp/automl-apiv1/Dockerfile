FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /automl-apiv1 ./examples/gcp/automl-apiv1
CMD ["/automl-apiv1"]
