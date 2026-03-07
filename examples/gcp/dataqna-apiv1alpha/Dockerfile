FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /dataqna-apiv1alpha ./examples/gcp/dataqna-apiv1alpha

ENTRYPOINT ["/dataqna-apiv1alpha"]
