FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /organizations ./examples/aws/organizations/organizations
ENTRYPOINT ["/organizations"]
