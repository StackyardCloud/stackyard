FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /geminidataanalytics-apiv1 ./examples/gcp/geminidataanalytics-apiv1

ENTRYPOINT ["/geminidataanalytics-apiv1"]
