FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /organizations-advanced ./examples/aws/organizations/organizations-advanced
ENTRYPOINT ["/organizations-advanced"]
