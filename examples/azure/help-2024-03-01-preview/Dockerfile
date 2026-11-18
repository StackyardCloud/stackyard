FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /help-2024-03-01-preview ./examples/azure/help-2024-03-01-preview

ENTRYPOINT ["/help-2024-03-01-preview"]
