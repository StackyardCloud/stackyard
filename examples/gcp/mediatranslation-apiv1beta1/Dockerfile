FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /mediatranslation-apiv1beta1 ./examples/gcp/mediatranslation-apiv1beta1

ENTRYPOINT ["/mediatranslation-apiv1beta1"]
