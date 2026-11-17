FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /powerbidedicated-2021-01-01 ./examples/azure/powerbidedicated-2021-01-01

ENTRYPOINT ["/powerbidedicated-2021-01-01"]
