FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /inspectorv2-basic ./examples/aws/inspectorv2/inspectorv2-basic
ENTRYPOINT ["/inspectorv2-basic"]
