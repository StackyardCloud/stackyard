FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /activedirectory-2021-04-01 ./examples/azure/activedirectory-2021-04-01

ENTRYPOINT ["/activedirectory-2021-04-01"]
