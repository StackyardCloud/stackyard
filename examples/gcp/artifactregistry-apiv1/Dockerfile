FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /artifactregistry-apiv1 ./examples/gcp/artifactregistry-apiv1
CMD ["/artifactregistry-apiv1"]
