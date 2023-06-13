oc apply -f configs/ (run twice)
oc new-build --name admission-controller --binary=true --strategy=docker -n admission-namespace
oc start-build admission-controller --from-dir=. --follow -n admission-namespace# cert-admission-ctrl
# cert-admission-ctrl-cert
# cert-admission-ctrl-cert
