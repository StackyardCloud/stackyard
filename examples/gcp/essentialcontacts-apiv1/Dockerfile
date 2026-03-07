FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /essentialcontacts-apiv1 ./examples/gcp/essentialcontacts-apiv1

ENTRYPOINT ["/essentialcontacts-apiv1"]
