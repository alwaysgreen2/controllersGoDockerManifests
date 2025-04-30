package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "strings"

    admissionv1 "k8s.io/api/admission/v1"
    appsv1      "k8s.io/api/apps/v1"
    metav1      "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func admitFunc(w http.ResponseWriter, r *http.Request) {
    var review admissionv1.AdmissionReview
    body, _ := ioutil.ReadAll(r.Body)
    _ = json.Unmarshal(body, &review)

    // Skip validation for exempt namespace
    if review.Request.Namespace == "webhook-demo" {
        response := admissionv1.AdmissionReview{
            TypeMeta: review.TypeMeta,
            Response: &admissionv1.AdmissionResponse{
                UID:     review.Request.UID,
                Allowed: true,
            },
        }
        respBytes, _ := json.Marshal(response)
        w.Header().Set("Content-Type", "application/json")
        w.Write(respBytes)
        return
    }

    dep := appsv1.Deployment{}
    _ = json.Unmarshal(review.Request.Object.Raw, &dep)

    var missing []string
    for _, c := range dep.Spec.Template.Spec.Containers {
        if c.LivenessProbe == nil {
            missing = append(missing, fmt.Sprintf("container %q missing liveness probe", c.Name))
        }
    }

    allowed := len(missing) == 0
    response := admissionv1.AdmissionReview{
        TypeMeta: review.TypeMeta,
        Response: &admissionv1.AdmissionResponse{
            UID:     review.Request.UID,
            Allowed: allowed,
        },
    }

    if !allowed {
        msg := "Liveness probes required:\n" + strings.Join(missing, "\n")
        response.Response.Result = &metav1.Status{Message: msg}
    }

    respBytes, _ := json.Marshal(response)
    w.Header().Set("Content-Type", "application/json")
    w.Write(respBytes)
}

func main() {
    http.HandleFunc("/validate", admitFunc)
    fmt.Println("Starting validating webhook on :8443")
    err := http.ListenAndServeTLS(":8443", "/tls/tls.crt", "/tls/tls.key", nil)
    if err != nil {
        fmt.Println("Failed to start server:", err)
    }
}
