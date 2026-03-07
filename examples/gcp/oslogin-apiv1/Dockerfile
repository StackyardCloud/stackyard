FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /oslogin-apiv1 ./examples/gcp/oslogin-apiv1

ENTRYPOINT ["/oslogin-apiv1"]
