FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /recaptchaenterprise-v2-apiv1 ./examples/gcp/recaptchaenterprise-v2-apiv1

ENTRYPOINT ["/recaptchaenterprise-v2-apiv1"]
