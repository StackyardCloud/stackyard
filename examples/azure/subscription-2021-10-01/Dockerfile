FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /subscription-2021-10-01 ./examples/azure/subscription-2021-10-01

ENTRYPOINT ["/subscription-2021-10-01"]
