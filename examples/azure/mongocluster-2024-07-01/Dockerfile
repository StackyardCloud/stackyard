FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /mongocluster-2024-07-01 ./examples/azure/mongocluster-2024-07-01

ENTRYPOINT ["/mongocluster-2024-07-01"]
