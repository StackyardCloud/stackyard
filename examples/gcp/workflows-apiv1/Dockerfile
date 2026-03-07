FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /workflows-apiv1 ./examples/gcp/workflows-apiv1

ENTRYPOINT ["/workflows-apiv1"]
