FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /workspaces-basic ./examples/aws/workspaces/workspaces-basic
ENTRYPOINT ["/workspaces-basic"]
