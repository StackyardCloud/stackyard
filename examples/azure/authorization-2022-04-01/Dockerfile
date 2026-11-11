FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /authorization-2022-04-01 ./examples/azure/authorization-2022-04-01

ENTRYPOINT ["/authorization-2022-04-01"]
