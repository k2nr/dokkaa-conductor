FROM crosbymichael/golang
MAINTAINER Kazunori Kajihiro <likerichie@gmail.com> (@k2nr)

ADD . /go/src/github.com/k2nr/dokkaa-scheduler/
RUN go get github.com/k2nr/dokkaa-scheduler

ENTRYPOINT ["dokkaa-scheduler"]
