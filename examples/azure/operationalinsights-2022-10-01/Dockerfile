FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /operationalinsights-2022-10-01 ./examples/azure/operationalinsights-2022-10-01

ENTRYPOINT ["/operationalinsights-2022-10-01"]
