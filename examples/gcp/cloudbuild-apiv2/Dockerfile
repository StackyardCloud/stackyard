FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /cloudbuild-apiv2 ./examples/gcp/cloudbuild-apiv2

ENTRYPOINT ["/cloudbuild-apiv2"]
