FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /resource-groups ./examples/aws/resource-groups
ENTRYPOINT ["/resource-groups"]
