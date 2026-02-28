FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /managedservices-cm-basic ./examples/managedservices-cm/managedservices-cm-basic
ENTRYPOINT ["/managedservices-cm-basic"]
