FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /policysimulator-apiv1 ./examples/gcp/policysimulator-apiv1

ENTRYPOINT ["/policysimulator-apiv1"]
