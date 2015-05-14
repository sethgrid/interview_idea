FROM golang:1.4.2

ENV GOPATH=/go \
    PROJECT_HOME=interview

# just working in /app for now
WORKDIR /$GOPATH/src/$PROJECT_HOME
ADD . /$GOPATH/src/$PROJECT_HOME

RUN go get -f -u
RUN go install

ENTRYPOINT ["interview"]
