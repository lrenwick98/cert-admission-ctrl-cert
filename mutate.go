package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/admission/v1"
	networking_v1 "k8s.io/api/networking/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// the struct the patch needs for each image
type patchOperation struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

var coreClient corev1.CoreV1Interface

func (srvstrc *myServerHandler) mutserve(w http.ResponseWriter, r *http.Request) {

	config, err := GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	coreClientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	coreClient := coreClientSet.CoreV1()

	ctx := context.TODO()

	timestampLog := Log()

	var patchData []patchOperation

	var Body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			Body = data
		}
	}
	if len(Body) == 0 {
		timestampLog.Errorf("Unable to retrieve body from API request")
		http.Error(w, "Empty Body", http.StatusBadRequest)
	}

	// Read the Response from the Kubernetes API and place it in the Request
	arRequest := &v1.AdmissionReview{}
	err = json.Unmarshal(Body, arRequest)
	if err != nil {
		timestampLog.Errorf("Error unmarshelling the body request")
		http.Error(w, "Error Unmarsheling the Body request", http.StatusBadRequest)
		return
	}

	raw := arRequest.Request.Object.Raw
	obj := networking_v1.Ingress{}

	err = json.Unmarshal(raw, &obj)
	if err != nil {
		timestampLog.Errorf("Error unmarshelling the body request")
		http.Error(w, "Error Unmarsheling the Body request", http.StatusBadRequest)
		return
	}

	svcList, err := coreClient.Services(obj.Namespace).List(ctx, meta_v1.ListOptions{})
	if err != nil {
		timestampLog.Errorf("Service list hasn't been retrieved")
	}

	for _, services := range svcList.Items {

		fmt.Print(services.ObjectMeta.Name)
	}

	arResponse := v1.AdmissionReview{
		Response: &v1.AdmissionResponse{
			UID: arRequest.Request.UID,
		},
	}

	if obj.Spec.Rules != nil {

		if obj.Spec.Rules[0].HTTP != nil {

			fullHostName := obj.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name + "-" + obj.Namespace + ".apps." + UrlSuffix.Spec.BaseDomain

			secretName := obj.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name + "-certificate"

			if obj.Spec.Rules[0].Host != fullHostName {
				patchData = append(patchData, patchOperation{
					Op:    "replace",
					Path:  "/spec/rules/0/host",
					Value: fullHostName})
			}
			if obj.Spec.TLS != nil {

				if obj.Spec.TLS[0].Hosts != nil {

					if obj.Spec.TLS[0].Hosts[0] != fullHostName {
						patchData = append(patchData, patchOperation{
							Op:    "replace",
							Path:  "/spec/tls/0/hosts/0",
							Value: fullHostName})
					}

					if obj.Spec.TLS[0].SecretName != secretName {
						patchData = append(patchData, patchOperation{
							Op:    "replace",
							Path:  "/spec/tls/0/secretName",
							Value: secretName})
					}
				}
			}
			if obj.ObjectMeta.Annotations != nil {
				patchData = append(patchData, patchOperation{
					Op:    "replace",
					Path:  "/metadata/annotations/" + "cert-manager.io~1issuer",
					Value: issuer})

				patchData = append(patchData, patchOperation{
					Op:    "replace",
					Path:  "/metadata/annotations/" + "cert-manager.io~1issuer-kind",
					Value: issuerKind})

				patchData = append(patchData, patchOperation{
					Op:    "replace",
					Path:  "/metadata/annotations/" + "cert-manager.io~1issuer-group",
					Value: issuerGroup})

				patchData = append(patchData, patchOperation{
					Op:    "replace",
					Path:  "/metadata/annotations/" + "cert-manager.io~1tls-acme",
					Value: "false"})

				patchData = append(patchData, patchOperation{
					Op:    "replace",
					Path:  "/metadata/annotations/" + "cert-manager.io~1common-name",
					Value: UrlSuffix.Spec.BaseDomain})
			}
		}
	}

	patchBytes, err := json.Marshal(patchData)
	if err != nil {
		timestampLog.Errorf("Can't encode response %v", err)
		http.Error(w, fmt.Sprintf("couldn't encode Patches: %v", err), http.StatusInternalServerError)
		return
	}
	v1JSONPatch := admissionv1.PatchTypeJSONPatch
	arResponse.APIVersion = "admission.k8s.io/v1"
	arResponse.Kind = arRequest.Kind
	arResponse.Response.Allowed = true
	arResponse.Response.Patch = patchBytes
	arResponse.Response.PatchType = &v1JSONPatch

	resp, err := json.Marshal(arResponse)
	if err != nil {
		timestampLog.Errorf("Can't encode response %v", err)
		http.Error(w, fmt.Sprintf("couldn't encode response: %v", err), http.StatusInternalServerError)
	}

	_, err = w.Write(resp)
	if err != nil {
		timestampLog.Errorf("Can't write response %v", err)
		http.Error(w, fmt.Sprintf("cloud not write response: %v", err), http.StatusInternalServerError)
	}
}
