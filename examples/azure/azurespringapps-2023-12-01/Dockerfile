FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /azurespringapps-2023-12-01 ./examples/azure/azurespringapps-2023-12-01

ENTRYPOINT ["/azurespringapps-2023-12-01"]
