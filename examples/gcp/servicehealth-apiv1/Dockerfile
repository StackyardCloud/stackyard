FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /servicehealth-apiv1 ./examples/gcp/servicehealth-apiv1

ENTRYPOINT ["/servicehealth-apiv1"]
