FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /rolesanywhere ./examples/aws/rolesanywhere
CMD ["/rolesanywhere"]
