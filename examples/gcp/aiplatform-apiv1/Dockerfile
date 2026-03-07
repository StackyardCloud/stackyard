FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /aiplatform-apiv1 ./examples/gcp/aiplatform-apiv1
CMD ["/aiplatform-apiv1"]
