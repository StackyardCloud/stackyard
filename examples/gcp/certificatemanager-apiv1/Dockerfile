FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /certificatemanager-apiv1 ./examples/gcp/certificatemanager-apiv1
CMD ["/certificatemanager-apiv1"]
