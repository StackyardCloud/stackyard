FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /rolesanywhere ./examples/aws/rolesanywhere/rolesanywhere
CMD ["/rolesanywhere"]
