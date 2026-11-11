FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /lifesciences-apiv2beta ./examples/gcp/lifesciences-apiv2beta

ENTRYPOINT ["/lifesciences-apiv2beta"]
