FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /organizations ./examples/aws/organizations
ENTRYPOINT ["/organizations"]
