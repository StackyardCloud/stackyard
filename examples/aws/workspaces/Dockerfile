FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /workspaces ./examples/aws/workspaces
ENTRYPOINT ["/workspaces"]
