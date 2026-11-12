FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /clouddms-apiv1 ./examples/gcp/clouddms-apiv1
CMD ["/clouddms-apiv1"]
