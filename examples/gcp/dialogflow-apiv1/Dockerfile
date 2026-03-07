FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /dialogflow-apiv1 ./examples/gcp/dialogflow-apiv1

ENTRYPOINT ["/dialogflow-apiv1"]
