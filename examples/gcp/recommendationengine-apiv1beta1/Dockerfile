FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /recommendationengine-apiv1beta1 ./examples/gcp/recommendationengine-apiv1beta1

ENTRYPOINT ["/recommendationengine-apiv1beta1"]
