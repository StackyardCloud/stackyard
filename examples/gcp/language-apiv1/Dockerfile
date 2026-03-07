FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /language-apiv1 ./examples/gcp/language-apiv1

ENTRYPOINT ["/language-apiv1"]

