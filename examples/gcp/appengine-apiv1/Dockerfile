FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /appengine-apiv1 ./examples/gcp/appengine-apiv1
CMD ["/appengine-apiv1"]
