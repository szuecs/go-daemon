FROM registry.opensource.zalan.do/stups/alpine:UPSTREAM

# add scm-source
ADD scm-source.json /

# add binary
ADD build/linux/go-daemon /

ENTRYPOINT ["/go-daemon"]
