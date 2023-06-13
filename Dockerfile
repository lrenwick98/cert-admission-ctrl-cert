FROM registry.redhat.io/ubi8/go-toolset AS builder
RUN mkdir -p /opt/app-root/src/admctrl
WORKDIR /opt/app-root/src/admctrl
ENV GOPATH=/opt/app-root/
ENV PATH="${PATH}:/opt/app-root/src/go/bin/"
COPY ./ .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o .
FROM ubi8/ubi-minimal
COPY --from=builder  /opt/app-root/src/admctrl/admission-controller-certificate /usr/bin/
USER 1001
EXPOSE 8080 8443
CMD ["/usr/bin/admission-controller-certificate"]
ENTRYPOINT ["/usr/bin/admission-controller-certificate"]

