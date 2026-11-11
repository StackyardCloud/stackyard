FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /azureintegration-2023-11-14-preview ./examples/azure/azureintegration-2023-11-14-preview

ENTRYPOINT ["/azureintegration-2023-11-14-preview"]
