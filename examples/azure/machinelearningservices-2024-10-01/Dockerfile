FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /machinelearningservices-2024-10-01 ./examples/azure/machinelearningservices-2024-10-01

ENTRYPOINT ["/machinelearningservices-2024-10-01"]
