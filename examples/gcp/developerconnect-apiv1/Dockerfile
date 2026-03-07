FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /developerconnect-apiv1 ./examples/gcp/developerconnect-apiv1

ENTRYPOINT ["/developerconnect-apiv1"]
