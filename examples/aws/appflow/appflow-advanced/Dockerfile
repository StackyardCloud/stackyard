FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /appflow-advanced ./examples/aws/appflow/appflow-advanced
ENTRYPOINT ["/appflow-advanced"]
