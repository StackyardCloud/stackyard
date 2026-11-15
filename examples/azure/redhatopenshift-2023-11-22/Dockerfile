FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /redhatopenshift-2023-11-22 ./examples/azure/redhatopenshift-2023-11-22

ENTRYPOINT ["/redhatopenshift-2023-11-22"]
