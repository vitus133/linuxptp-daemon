FROM golang:1.19 AS builder
WORKDIR /go/src/github.com/openshift/linuxptp-daemon
COPY . .
RUN make clean && make

FROM quay.io/openshift/origin-base:4.13
RUN yum -y update && yum --setopt=skip_missing_names_on_install=False -y install ethtool make gcc hwdata strace && yum clean all
COPY --from=builder /go/src/github.com/openshift/linuxptp-daemon/bin/ptp /usr/local/bin/
COPY ./extra/leap-seconds.list /usr/share/zoneinfo/leap-seconds.list
COPY ./linuxptp /linuxptp
WORKDIR /linuxptp
RUN make && make install && echo '[global]' > /etc/ptp4l.conf && mv /usr/local/sbin/* /usr/sbin/

CMD ["/usr/local/bin/ptp"]
