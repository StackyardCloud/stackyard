FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /workspaces-web ./examples/aws/workspaces-web
ENTRYPOINT ["/workspaces-web"]
