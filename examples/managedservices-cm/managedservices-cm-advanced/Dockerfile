FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /managedservices-cm-advanced ./examples/managedservices-cm/managedservices-cm-advanced
ENTRYPOINT ["/managedservices-cm-advanced"]
