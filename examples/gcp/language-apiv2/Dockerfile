FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /language-apiv2 ./examples/gcp/language-apiv2

ENTRYPOINT ["/language-apiv2"]

