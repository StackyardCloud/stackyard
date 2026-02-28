FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /incident-manager-basic ./examples/incident-manager/incident-manager-basic
ENTRYPOINT ["/incident-manager-basic"]
