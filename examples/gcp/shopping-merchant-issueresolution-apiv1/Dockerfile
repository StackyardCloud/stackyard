FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /shopping-merchant-issueresolution-apiv1 ./examples/gcp/shopping-merchant-issueresolution-apiv1
CMD ["/shopping-merchant-issueresolution-apiv1"]
