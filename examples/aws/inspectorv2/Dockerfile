FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /inspectorv2 ./examples/aws/inspectorv2
ENTRYPOINT ["/inspectorv2"]
