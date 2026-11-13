FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /iothub-2023-06-30 ./examples/azure/iothub-2023-06-30

ENTRYPOINT ["/iothub-2023-06-30"]
