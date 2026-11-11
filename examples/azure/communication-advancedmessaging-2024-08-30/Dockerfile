FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /communication-advancedmessaging-2024-08-30 ./examples/azure/communication-advancedmessaging-2024-08-30

ENTRYPOINT ["/communication-advancedmessaging-2024-08-30"]
