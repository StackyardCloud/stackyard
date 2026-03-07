FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /licensemanager-apiv1 ./examples/gcp/licensemanager-apiv1

ENTRYPOINT ["/licensemanager-apiv1"]
