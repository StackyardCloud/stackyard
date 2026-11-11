FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /arc-vmware-2023-12-01 ./examples/azure/arc-vmware-2023-12-01

ENTRYPOINT ["/arc-vmware-2023-12-01"]
