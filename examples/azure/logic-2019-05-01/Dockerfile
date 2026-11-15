FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /logic-2019-05-01 ./examples/azure/logic-2019-05-01

ENTRYPOINT ["/logic-2019-05-01"]
