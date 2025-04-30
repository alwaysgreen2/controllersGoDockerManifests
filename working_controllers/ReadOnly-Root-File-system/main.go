package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    admissionv1 "k8s.io/api/admission/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func admitFunc(w http.ResponseWriter, r *http.Request) {
    var review admissionv1.AdmissionReview
    body, _ := ioutil.ReadAll(r.Body)
    _ = json.Unmarshal(body, &review)

    allowed := true
    var result *metav1.Status = nil

    var pod struct {
        Spec struct {
            Containers []struct {
                SecurityContext *struct {
                    ReadOnlyRootFilesystem *bool `json:"readOnlyRootFilesystem"`
                } `json:"securityContext"`
            } `json:"containers"`
        } `json:"spec"`
    }
    _ = json.Unmarshal(review.Request.Object.Raw, &pod)

    for _, container := range pod.Spec.Containers {
        if container.SecurityContext == nil || container.SecurityContext.ReadOnlyRootFilesystem == nil || !*container.SecurityContext.ReadOnlyRootFilesystem {
            allowed = false
            result = &metav1.Status{
                Message: "All containers must set securityContext.readOnlyRootFilesystem=true.",
            }
            break
        }
    }

    response := admissionv1.AdmissionReview{
        TypeMeta: review.TypeMeta,
        Response: &admissionv1.AdmissionResponse{
            UID:     review.Request.UID,
            Allowed: allowed,
            Result:  result,
        },
    }
    respBytes, _ := json.Marshal(response)
    w.Header().Set("Content-Type", "application/json")
    w.Write(respBytes)
}

func main() {
    http.HandleFunc("/validate", admitFunc)
    fmt.Println("Starting webhook server on :8443")
    err := http.ListenAndServeTLS(":8443", "/tls/tls.crt", "/tls/tls.key", nil)
    if err != nil {
        fmt.Println("Failed to start server:", err)
    }
}
