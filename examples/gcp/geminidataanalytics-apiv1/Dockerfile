FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /geminidataanalytics-apiv1 ./examples/gcp/geminidataanalytics-apiv1

ENTRYPOINT ["/geminidataanalytics-apiv1"]
