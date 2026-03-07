FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /policytroubleshooter-iam-apiv3 ./examples/gcp/policytroubleshooter-iam-apiv3

ENTRYPOINT ["/policytroubleshooter-iam-apiv3"]
