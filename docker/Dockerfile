FROM scratch
MAINTAINER cblomart@gmail.com

COPY ./spliceproxy /spliceproxy
COPY ./config.sample.yaml /etc/config.yaml

VOLUME [ "/etc" ]

ENTRYPOINT [ "/spliceproxy", "-c", "/etc/config.yaml" ] 