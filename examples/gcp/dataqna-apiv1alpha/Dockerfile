FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /dataqna-apiv1alpha ./examples/gcp/dataqna-apiv1alpha

ENTRYPOINT ["/dataqna-apiv1alpha"]
