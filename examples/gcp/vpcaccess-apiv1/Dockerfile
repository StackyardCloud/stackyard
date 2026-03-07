FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /vpcaccess-apiv1 ./examples/gcp/vpcaccess-apiv1

ENTRYPOINT ["/vpcaccess-apiv1"]
