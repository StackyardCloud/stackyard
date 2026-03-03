FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /deadline-cloud-basic ./examples/aws/deadline-cloud/deadline-cloud-basic
ENTRYPOINT ["/deadline-cloud-basic"]
