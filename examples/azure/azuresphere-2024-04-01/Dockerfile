FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /azuresphere-2024-04-01 ./examples/azure/azuresphere-2024-04-01

ENTRYPOINT ["/azuresphere-2024-04-01"]
