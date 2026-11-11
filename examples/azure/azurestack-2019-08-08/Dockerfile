FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /azurestack-2019-08-08 ./examples/azure/azurestack-2019-08-08

ENTRYPOINT ["/azurestack-2019-08-08"]
