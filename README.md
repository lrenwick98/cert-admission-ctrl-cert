Certificate admission controller which validates for Certificate objects to follow a specific naming convention alongside other checks

- Host name and common name are validated so they follow this naming convention: <service>-<namespace>.apps.<ClusterBaseDomain>
- Services within the namespace retrieved are checked against the Certificate object to ensure user is attaching a valid service into the object
- DNS name is checked to ensure only 1 entry exists
- DNS name and Common name are checked to ensure they reflect each other

oc apply -f configs/ (run twice)
oc new-build --name admission-controller-certificate --binary=true --strategy=docker -n admission-namespace
oc start-build admission-controller-certificate --from-dir=. --follow -n admission-namespace# cert-admission-ctrl
# cert-admission-ctrl-certificate
# cert-admission-ctrl-certificate
