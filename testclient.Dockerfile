FROM golang:1.8.3

RUN apt-get update && apt-get install -y \
	python3 \
	python3-pip \
	--no-install-recommends \
	&& rm -rf /var/lib/apt/lists/* 

RUN pip3 install requests

ENV NOTARYDIR /go/src/github.com/docker/notary

COPY . ${NOTARYDIR}
RUN chmod -R a+rw /go

WORKDIR ${NOTARYDIR}

ENTRYPOINT ["python3", "./buildscripts/testchangefeed.py"]