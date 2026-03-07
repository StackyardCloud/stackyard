FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /artifactregistry-apiv1 ./examples/gcp/artifactregistry-apiv1
CMD ["/artifactregistry-apiv1"]
