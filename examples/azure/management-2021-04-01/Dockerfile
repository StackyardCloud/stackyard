FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /management-2021-04-01 ./examples/azure/management-2021-04-01

ENTRYPOINT ["/management-2021-04-01"]
