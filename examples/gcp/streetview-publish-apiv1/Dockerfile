FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /streetview-publish-apiv1 ./examples/gcp/streetview-publish-apiv1

ENTRYPOINT ["/streetview-publish-apiv1"]
