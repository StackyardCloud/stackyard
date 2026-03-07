FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /clouddms-apiv1 ./examples/gcp/clouddms-apiv1
CMD ["/clouddms-apiv1"]
