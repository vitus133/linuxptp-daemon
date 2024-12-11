FROM registry.ci.openshift.org/ocp/builder:rhel-9-golang-1.20-openshift-4.14 AS builder
WORKDIR /go/src/github.com/openshift/linuxptp-daemon
COPY . .
RUN make clean && make

FROM registry.ci.openshift.org/ocp/4.14:base-rhel9

COPY ./extra/leap-seconds.list /usr/share/zoneinfo/leap-seconds.list
COPY ./extra/linuxptp-3.1.1-6.el9_2.7.x86_64.rpm /linuxptp-3.1.1-6.el9_2.7.x86_64.rpm

RUN yum -y update && yum -y update glibc && yum --setopt=skip_missing_names_on_install=False -y install ethtool hwdata  && yum clean all
RUN yum install -y gpsd-minimal
RUN yum install -y gpsd-minimal-clients

# Create symlinks for executables to match references
RUN ln -s /usr/bin/gpspipe /usr/local/bin/gpspipe
RUN ln -s /usr/sbin/gpsd /usr/local/sbin/gpsd
RUN ln -s /usr/bin/ubxtool /usr/local/bin/ubxtool

RUN yum install -y /linuxptp-3.1.1-6.el9_2.7.x86_64.rpm

COPY --from=builder /go/src/github.com/openshift/linuxptp-daemon/bin/ptp /usr/local/bin/

CMD ["/usr/local/bin/ptp"]
