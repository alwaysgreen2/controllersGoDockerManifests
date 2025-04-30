package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"

    admissionv1 "k8s.io/api/admission/v1"
    appsv1 "k8s.io/api/apps/v1"
)

func admitFunc(w http.ResponseWriter, r *http.Request) {
    var review admissionv1.AdmissionReview
    body, _ := ioutil.ReadAll(r.Body)
    _ = json.Unmarshal(body, &review)

    deployment := appsv1.Deployment{}
    _ = json.Unmarshal(review.Request.Object.Raw, &deployment)

    patch := []byte("[]")
    allowed := true

    excludedNamespace := os.Getenv("EXCLUDED_NAMESPACE")
    if excludedNamespace == "" {
        excludedNamespace = "webhook-demo"
    }

    if review.Request.Kind.Kind == "Deployment" && review.Request.Namespace != excludedNamespace {
        replicas := int32(1)
        if deployment.Spec.Replicas != nil {
            replicas = *deployment.Spec.Replicas
        }
        if replicas < 3 {
            patch = []byte(`[
                {"op": "replace", "path": "/spec/replicas", "value": 3}
            ]`)
        }
    }

    response := admissionv1.AdmissionReview{
        TypeMeta: review.TypeMeta,
        Response: &admissionv1.AdmissionResponse{
            UID:     review.Request.UID,
            Allowed: allowed,
            Patch:   patch,
            PatchType: func() *admissionv1.PatchType {
                pt := admissionv1.PatchTypeJSONPatch
                return &pt
            }(),
        },
    }

    respBytes, _ := json.Marshal(response)
    w.Header().Set("Content-Type", "application/json")
    w.Write(respBytes)
}

func main() {
    http.HandleFunc("/mutate", admitFunc)
    fmt.Println("Starting webhook server on :8443")
    err := http.ListenAndServeTLS(":8443", "/tls/tls.crt", "/tls/tls.key", nil)
    if err != nil {
        fmt.Println("Failed to start server:", err)
    }
}
