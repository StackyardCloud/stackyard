FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /desktopvirtualization-2025-10-10-example ./examples/azure/desktopvirtualization-2025-10-10

ENTRYPOINT ["/desktopvirtualization-2025-10-10-example"]
