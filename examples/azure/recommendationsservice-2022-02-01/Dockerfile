FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /recommendationsservice-2022-02-01 ./examples/azure/recommendationsservice-2022-02-01

ENTRYPOINT ["/recommendationsservice-2022-02-01"]
