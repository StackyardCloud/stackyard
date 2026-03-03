FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /deadline-cloud ./examples/aws/deadline-cloud/deadline-cloud
ENTRYPOINT ["/deadline-cloud"]
