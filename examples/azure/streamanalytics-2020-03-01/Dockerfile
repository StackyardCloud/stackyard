FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /streamanalytics-2020-03-01 ./examples/azure/streamanalytics-2020-03-01

ENTRYPOINT ["/streamanalytics-2020-03-01"]
