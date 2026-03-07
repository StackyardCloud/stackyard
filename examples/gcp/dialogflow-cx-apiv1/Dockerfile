FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /dialogflow-cx-apiv1 ./examples/gcp/dialogflow-cx-apiv1

ENTRYPOINT ["/dialogflow-cx-apiv1"]
