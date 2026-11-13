FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /hdinsight-2021-06-01 ./examples/azure/hdinsight-2021-06-01

ENTRYPOINT ["/hdinsight-2021-06-01"]
