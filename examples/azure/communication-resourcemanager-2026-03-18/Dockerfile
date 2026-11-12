FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /communication-resourcemanager-2026-03-18 ./examples/azure/communication-resourcemanager-2026-03-18

ENTRYPOINT ["/communication-resourcemanager-2026-03-18"]
