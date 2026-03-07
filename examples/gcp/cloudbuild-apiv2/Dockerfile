FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /cloudbuild-apiv2 ./examples/gcp/cloudbuild-apiv2

ENTRYPOINT ["/cloudbuild-apiv2"]
