FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /resourcemover-2023-08-01 ./examples/azure/resourcemover-2023-08-01

ENTRYPOINT ["/resourcemover-2023-08-01"]
