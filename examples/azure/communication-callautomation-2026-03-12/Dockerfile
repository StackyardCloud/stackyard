FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /communication-callautomation-2026-03-12 ./examples/azure/communication-callautomation-2026-03-12

ENTRYPOINT ["/communication-callautomation-2026-03-12"]
