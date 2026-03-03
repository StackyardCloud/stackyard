FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /appflow-basic ./examples/aws/appflow/appflow-basic
ENTRYPOINT ["/appflow-basic"]
