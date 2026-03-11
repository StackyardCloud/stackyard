FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /analysis-services-2017-08-01 ./examples/azure/analysis-services-2017-08-01

ENTRYPOINT ["/analysis-services-2017-08-01"]
