FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /workspaces-core ./examples/aws/workspaces-core
ENTRYPOINT ["/workspaces-core"]
