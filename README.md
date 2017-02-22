# rai-project/docker

Docker uses [rai-project/config](https://github.com/rai-project/config) for configuration.
The docker section of the configuration file should be written according to [config.go]().
# Docker [![Build Status](https://travis-ci.org/rai-project/docker.svg?branch=master)](https://travis-ci.org/rai-project/docker)

## Config

~~~
docker:
  - time_limit: 1h
  - image: ubuntu
  - memory_limit: 4gb
  - endpoints:
    - /run/docker.sock
    - /var/run/docker.sock
  - env:
    CI: rai
    RAI: true
    RAI_ARCH: linux/amd64
    RAI_USER: root
    RAI_SOURCE_DIR: /src
    RAI_DATA_DIR: /data
    RAI_BUILD_DIR: /build
    SOURCE_DIR: /src 
    DATA_DIR: /data 
    BUILD_DIR: /build 
    PATH: /usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
    TERM: xterm

~~~
