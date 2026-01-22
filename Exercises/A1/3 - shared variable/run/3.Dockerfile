FROM ubuntu:24.04 

RUN apt-get update && \
    apt-get install -y build-essential gcc golang && \
    rm -rf /var/lib/apt/lists/*

ENV GOPATH=/go
ENV PATH=$PATH:/usr/lib/go-1.21/bin:$GOPATH/bin

WORKDIR /workspace

CMD ["/bin/bash"]
