FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /datalakeanalytics-example ./examples/azure/datalakeanalytics-2016-11-01

ENTRYPOINT ["/datalakeanalytics-example"]
