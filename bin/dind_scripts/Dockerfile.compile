FROM k8s.gcr.io/debian-base-amd64:0.3

RUN apt-get update && apt-get install -y \
        automake \
        autotools-dev \
        bash \
        build-essential \
        curl \
        git-core \
        libcurl4-openssl-dev \
        libfuse-dev \
        libssl-dev \
        libxml2-dev \
        pkg-config \
        wget vim && \
     rm -rf /var/lib/apt/lists/*

# Install Go
RUN wget https://dl.google.com/go/go1.10.1.linux-amd64.tar.gz && \
    tar -xvf go1.10.1.linux-amd64.tar.gz && mv go /usr/local && mkdir -p ~/go/bin

ENV GOROOT=/usr/local/go
ENV GOPATH=/root/go
ENV PATH="${GOPATH}/bin:${GOROOT}/bin:${PATH}"

# Install glide
RUN curl https://glide.sh/get | sh && go get github.com/gin-gonic/gin

# Compile s3fs-fuse
RUN git clone https://github.com/s3fs-fuse/s3fs-fuse.git && \
    cd s3fs-fuse && ./autogen.sh && ./configure CPPFLAGS='-I/usr/local/opt/openssl/include' && make

# Compile storage plugin
RUN mkdir -p ${GOPATH}/bin && mkdir -p ${GOPATH}/src/github.com/IBM && \
    cd ${GOPATH}/src/github.com/IBM && git clone https://github.com/IBM/ibmcloud-object-storage-plugin.git && \
    cd ibmcloud-object-storage-plugin && make
#    cd ibmcloud-object-storage-plugin && make && make provisioner && make driver

# Note: The following lines were for Ubuntu, need to migrate to Debian if you should want to use them
#RUN apt-get update && apt-get install -y \
#            apt-transport-https \
#            software-properties-common && \
#    curl -fsSL https://download.docker.com/linux/debian/gpg | apt-key add - && \
#    add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/debian $(lsb_release -cs) stable" && \
#    apt-get update && apt-get install -y docker-ce && \
#    curl -L https://github.com/docker/compose/releases/download/1.18.0/docker-compose-`uname -s`-`uname -m` -o /usr/local/bin/docker-compose && \
#    rm -rf /var/lib/apt/lists/*
