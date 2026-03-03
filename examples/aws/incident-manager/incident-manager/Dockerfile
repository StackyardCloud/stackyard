FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /incident-manager ./examples/aws/incident-manager/incident-manager
ENTRYPOINT ["/incident-manager"]
