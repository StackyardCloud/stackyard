FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /workspaces-core-advanced ./examples/aws/workspaces-core/workspaces-core-advanced
ENTRYPOINT ["/workspaces-core-advanced"]
