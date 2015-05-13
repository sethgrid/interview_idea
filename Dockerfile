FROM golang:1.4.2

ENV GOPATH=/app:/app/vendor GOBIN=/usr/local/bin
WORKDIR /app
ADD . /app
RUN go install interview

ENTRYPOINT ["interview"]
