FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /workspaces-advanced ./examples/workspaces/workspaces-advanced
ENTRYPOINT ["/workspaces-advanced"]
