FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /workspaces-core-basic ./examples/aws/workspaces-core/workspaces-core-basic
ENTRYPOINT ["/workspaces-core-basic"]
