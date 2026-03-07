FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /cloudprofiler-apiv2 ./examples/gcp/cloudprofiler-apiv2

ENTRYPOINT ["/cloudprofiler-apiv2"]
