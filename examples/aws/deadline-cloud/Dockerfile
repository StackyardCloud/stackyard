FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /deadline-cloud ./examples/aws/deadline-cloud
ENTRYPOINT ["/deadline-cloud"]
