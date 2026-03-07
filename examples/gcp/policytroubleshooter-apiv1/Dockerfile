FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /policytroubleshooter-apiv1 ./examples/gcp/policytroubleshooter-apiv1

ENTRYPOINT ["/policytroubleshooter-apiv1"]
