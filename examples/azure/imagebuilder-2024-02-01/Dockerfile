FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /imagebuilder-2024-02-01 ./examples/azure/imagebuilder-2024-02-01

ENTRYPOINT ["/imagebuilder-2024-02-01"]
