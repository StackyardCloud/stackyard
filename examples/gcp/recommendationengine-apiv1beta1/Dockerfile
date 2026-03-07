FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /recommendationengine-apiv1beta1 ./examples/gcp/recommendationengine-apiv1beta1

ENTRYPOINT ["/recommendationengine-apiv1beta1"]
