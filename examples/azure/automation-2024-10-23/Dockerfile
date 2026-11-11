FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /automation-2024-10-23 ./examples/azure/automation-2024-10-23

ENTRYPOINT ["/automation-2024-10-23"]
