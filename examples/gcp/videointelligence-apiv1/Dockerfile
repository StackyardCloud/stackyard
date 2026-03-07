FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /videointelligence-apiv1 ./examples/gcp/videointelligence-apiv1

ENTRYPOINT ["/videointelligence-apiv1"]
