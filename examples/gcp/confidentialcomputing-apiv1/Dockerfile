FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /confidentialcomputing-apiv1 ./examples/gcp/confidentialcomputing-apiv1

ENTRYPOINT ["/confidentialcomputing-apiv1"]
