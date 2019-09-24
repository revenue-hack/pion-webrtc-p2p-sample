FROM golang:1.12

WORKDIR /app

COPY ./go.mod ./go.sum /app/
RUN go get

COPY ./main.go  /app/main.go
RUN go build -o offer
CMD ["/app/offer"]

