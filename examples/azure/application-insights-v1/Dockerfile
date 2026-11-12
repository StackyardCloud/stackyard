FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /application-insights-v1 ./examples/azure/application-insights-v1

ENTRYPOINT ["/application-insights-v1"]
