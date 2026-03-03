FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /workspaces-core ./examples/aws/workspaces-core/workspaces-core
ENTRYPOINT ["/workspaces-core"]
