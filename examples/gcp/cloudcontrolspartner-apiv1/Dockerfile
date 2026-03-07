FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudcontrolspartner-apiv1 ./examples/gcp/cloudcontrolspartner-apiv1
CMD ["/cloudcontrolspartner-apiv1"]
