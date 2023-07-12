package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	v1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// Creation of validate  function to be used in main.go
func (gs *myServerHandler) valserve(w http.ResponseWriter, r *http.Request) {

	timestampLog := Log()

	var coreClient corev1.CoreV1Interface

	var Body []byte

	var errorMessage string
	// Declaring of boolean variables so conditional checks can be validated against later
	serviceCorrect := false
	namespaceCorrect := false
	appsCorrect := false
	baseDomainCorrect := false
	dnsLenCorrect := false
	SANequalsCN := false
	// Retrieval of auth token so logic can be executed within OCP
	config, err := GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	coreClientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	coreClient = coreClientSet.CoreV1()

	ctx := context.TODO()
	// Error handling of reading body of request
	if r.Body != nil {
		data, err := ioutil.ReadAll(r.Body)
		if err == nil {
			Body = data
		}
	}
	// Error handling for not retrieving any data
	if len(Body) == 0 {
		timestampLog.Errorf("Unable to retrieve Body from webhook: %v", http.StatusBadRequest)
		http.Error(w, "Unable to retrieve Body from the API", http.StatusBadRequest)
		return
	}

	arRequest := &admissionv1.AdmissionReview{}
	// Error handling for unmarshalling request
	err = json.Unmarshal(Body, arRequest)
	if err != nil {
		timestampLog.Errorf("Unable to unmarshal the request: %v", http.StatusBadRequest)
		http.Error(w, "unable to Unmarshal the Body Request", http.StatusBadRequest)
		return
	}

	// initial the POD values from the request
	raw := arRequest.Request.Object.Raw
	obj := v1.Certificate{}

	if err := json.Unmarshal(raw, &obj); err != nil {
		timestampLog.Errorf("Unable to unmarshal the request: %v", http.StatusBadRequest)
		http.Error(w, "Unable to Unmarshal the Pod Information", http.StatusBadRequest)
	}

	// Creating of variable which retrieves list of services in given namespace

	svcList, err := coreClient.Services(obj.Namespace).List(ctx, metav1.ListOptions{})
	// For loop for checking if service within given namespace exists within the DNS name of Certificate object
	for _, services := range svcList.Items {
		for _, dnsName := range obj.Spec.DNSNames {
			if strings.Contains(dnsName, services.ObjectMeta.Name) {
				serviceCorrect = true
			} else {
				errorMessage = "Make sure DNS & Common name name follows this format: <service>-<namespace>.apps." + UrlSuffix.Spec.BaseDomain
			}
		}
	}
	// namespace check
	for _, dnsName := range obj.Spec.DNSNames {
		if strings.Contains(dnsName, obj.Namespace) {
			namespaceCorrect = true
		} else {
			errorMessage = "Make sure DNS & Common name follows this format: <service>-<namespace>.apps." + UrlSuffix.Spec.BaseDomain
		}
	}
	// basedomain check
	for _, dnsName := range obj.Spec.DNSNames {
		if strings.Contains(dnsName, UrlSuffix.Spec.BaseDomain) {
			baseDomainCorrect = true
		} else {
			errorMessage = "Make sure DNS & Common name follows this format: <service>-<namespace>.apps." + UrlSuffix.Spec.BaseDomain
		}
	}
	// .apps. check
	for _, dnsName := range obj.Spec.DNSNames {
		if strings.Contains(dnsName, ".apps.") {
			appsCorrect = true
		} else {
			errorMessage = "Make sure DNS & Common name follows this format: <service>-<namespace>.apps." + UrlSuffix.Spec.BaseDomain
		}
	}
	// check length of dns names = 1
	dnsLen := len(obj.Spec.DNSNames)

	if dnsLen == 1 {
		dnsLenCorrect = true
	} else {
		errorMessage = "Make sure there is only one DNS name and follows the same structure of Common Name"
	}

	// fmt.Print("\n")

	// fmt.Print(obj.Spec.DNSNames[0])

	// fmt.Print("\n")

	// fmt.Print(obj.Spec.CommonName)

	// fmt.Print("\n")

	// Check if DNS name matches with Common Name
	if (obj.Spec.DNSNames[0]) == (obj.Spec.CommonName) {
		SANequalsCN = true
	} else {
		errorMessage = "Make sure DNS and Common Name values are the same"
	}
	// Building admission review response to send back to server
	arResponse := admissionv1.AdmissionReview{
		Response: &admissionv1.AdmissionResponse{
			Result:  &metav1.Status{Status: "Failure", Message: errorMessage, Code: 406},
			UID:     arRequest.Request.UID,
			Allowed: false,
		},
	}
	// Check if conditional variables set are all satisfied, if so return validation success back to server to allow request
	arResponse.APIVersion = "admission.k8s.io/v1"
	arResponse.Kind = arRequest.Kind
	if serviceCorrect && namespaceCorrect && baseDomainCorrect && appsCorrect && dnsLenCorrect && SANequalsCN {
		arResponse.Response.Allowed = true
		arResponse.Response.Result = &metav1.Status{Status: "Success",
			Message: "All conditions have been met and are validated",
			Code:    201}
	}
	// Error handling for marshalling and writing back response to server
	resp, err := json.Marshal(arResponse)
	if err != nil {
		timestampLog.Errorf("Unable to marshal the request: %v", http.StatusBadRequest)
		http.Error(w, "Unable to Marshal the Request", http.StatusBadRequest)
	}

	_, err = w.Write(resp)
	if err != nil {
		timestampLog.Errorf("Unable to write the response, HTTP error: %v", http.StatusBadRequest)
		http.Error(w, "Unable to Write Response", http.StatusBadRequest)
	}
}
