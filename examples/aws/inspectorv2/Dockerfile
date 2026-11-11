FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /inspectorv2 ./examples/aws/inspectorv2
ENTRYPOINT ["/inspectorv2"]
