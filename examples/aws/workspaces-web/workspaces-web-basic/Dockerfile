FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /workspaces-web-basic ./examples/aws/workspaces-web/workspaces-web-basic
ENTRYPOINT ["/workspaces-web-basic"]
