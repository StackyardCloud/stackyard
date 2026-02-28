FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /organizations-basic ./examples/organizations/organizations-basic
ENTRYPOINT ["/organizations-basic"]
