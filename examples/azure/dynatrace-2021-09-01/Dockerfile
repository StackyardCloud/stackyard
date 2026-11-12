FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /dynatrace-2021-09-01-example ./examples/azure/dynatrace-2021-09-01

ENTRYPOINT ["/dynatrace-2021-09-01-example"]

