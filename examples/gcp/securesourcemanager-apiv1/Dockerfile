FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /securesourcemanager-apiv1 ./examples/gcp/securesourcemanager-apiv1

ENTRYPOINT ["/securesourcemanager-apiv1"]
