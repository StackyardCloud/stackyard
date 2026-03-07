FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /apihub-apiv1 ./examples/gcp/apihub-apiv1
CMD ["/apihub-apiv1"]
