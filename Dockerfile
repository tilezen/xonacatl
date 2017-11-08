FROM golang:1.8

WORKDIR /go/src/app
COPY . .

RUN go get -u github.com/golang/protobuf/proto \
 && go get -u github.com/tilezen/xonacatl/xonacatl_server \
 && go install github.com/tilezen/xonacatl/xonacatl_server

ENV XONACATL_LISTEN=":8000"
EXPOSE 8000

CMD ["xonacatl_server"]
