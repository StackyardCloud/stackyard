FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /recommender-apiv1 ./examples/gcp/recommender-apiv1

ENTRYPOINT ["/recommender-apiv1"]
