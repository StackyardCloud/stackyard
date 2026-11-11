FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /dialogflow-apiv1 ./examples/gcp/dialogflow-apiv1

ENTRYPOINT ["/dialogflow-apiv1"]
