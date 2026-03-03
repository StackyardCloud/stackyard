FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /rolesanywhere-basic ./examples/aws/rolesanywhere/rolesanywhere-basic
CMD ["/rolesanywhere-basic"]
