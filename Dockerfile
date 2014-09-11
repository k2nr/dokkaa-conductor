FROM crosbymichael/golang
MAINTAINER Kazunori Kajihiro <likerichie@gmail.com> (@k2nr)

ADD . /go/src/github.com/k2nr/dokkaa-conductor/
RUN go get github.com/k2nr/dokkaa-conductor

ENTRYPOINT ["dokkaa-conductor"]
