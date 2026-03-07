FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /generativelanguage-apiv1 ./examples/gcp/generativelanguage-apiv1
CMD ["/generativelanguage-apiv1"]
