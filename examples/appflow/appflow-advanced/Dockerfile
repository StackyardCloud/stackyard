FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /appflow-advanced ./examples/appflow/appflow-advanced
ENTRYPOINT ["/appflow-advanced"]
