FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /generativelanguage-apiv1-basic ./examples/gcp/generativelanguage-apiv1/generativelanguage-apiv1-basic
CMD ["/generativelanguage-apiv1-basic"]
