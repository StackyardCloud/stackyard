FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /deadline-cloud-advanced ./examples/deadline-cloud/deadline-cloud-advanced
ENTRYPOINT ["/deadline-cloud-advanced"]
