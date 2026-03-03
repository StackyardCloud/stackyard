FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /incident-manager-advanced ./examples/aws/incident-manager/incident-manager-advanced
ENTRYPOINT ["/incident-manager-advanced"]
