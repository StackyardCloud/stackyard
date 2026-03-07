FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /confidentialcomputing-apiv1 ./examples/gcp/confidentialcomputing-apiv1

ENTRYPOINT ["/confidentialcomputing-apiv1"]
