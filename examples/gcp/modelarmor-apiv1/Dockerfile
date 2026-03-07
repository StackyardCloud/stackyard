FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /modelarmor-apiv1 ./examples/gcp/modelarmor-apiv1

ENTRYPOINT ["/modelarmor-apiv1"]
