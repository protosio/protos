#
# Git
# 
# VERSION 0.1
# 

FROM ubuntu
MAINTAINER Alex Giurgiu "alex@giurgiu.io"
EXPOSE 2222:22

ADD ./bootstrap /tmp/bootstrap/
RUN chmod +x /tmp/bootstrap/bootstrap.sh && /tmp/bootstrap/bootstrap.sh
