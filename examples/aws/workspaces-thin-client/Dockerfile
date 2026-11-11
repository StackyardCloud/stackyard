FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /workspaces-thin-client ./examples/aws/workspaces-thin-client
ENTRYPOINT ["/workspaces-thin-client"]
