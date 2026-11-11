FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /datalabeling-apiv1 ./examples/gcp/datalabeling-apiv1

ENTRYPOINT ["/datalabeling-apiv1"]
