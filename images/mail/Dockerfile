#
# Mail 
# 
# VERSION 0.1
# 

FROM centos
MAINTAINER Alex Giurgiu "alex@giurgiu.io"
EXPOSE 25:25 993:993

ADD ./bootstrap /tmp/bootstrap/
RUN chmod +x /tmp/bootstrap/bootstrap.sh && /tmp/bootstrap/bootstrap.sh

