FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /managedservices-cm ./examples/aws/managedservices-cm
ENTRYPOINT ["/managedservices-cm"]
