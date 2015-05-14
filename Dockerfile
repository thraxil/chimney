FROM golang
ADD . /go/src/github.com/thraxil/chimney
RUN go install github.com/thraxil/chimney
RUN mkdir /etc/chimney
ENTRYPOINT /go/bin/chimney -config=/etc/chimney/config.json
