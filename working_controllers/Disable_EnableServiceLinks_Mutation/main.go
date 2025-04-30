package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"

    admissionv1 "k8s.io/api/admission/v1"
    appsv1      "k8s.io/api/apps/v1"
)

func mutateFunc(w http.ResponseWriter, r *http.Request) {
    var review admissionv1.AdmissionReview
    body, _ := ioutil.ReadAll(r.Body)
    _ = json.Unmarshal(body, &review)

    dep := appsv1.Deployment{}
    _ = json.Unmarshal(review.Request.Object.Raw, &dep)

    var patches []map[string]interface{}

    // If enableServiceLinks is unset or true, patch it to false
    if dep.Spec.Template.Spec.EnableServiceLinks == nil || *dep.Spec.Template.Spec.EnableServiceLinks {
        patches = append(patches, map[string]interface{}{
            "op":    "add",
            "path":  "/spec/template/spec/enableServiceLinks",
            "value": false,
        })
    }

    patchBytes, _ := json.Marshal(patches)
    pt := admissionv1.PatchTypeJSONPatch
    response := admissionv1.AdmissionResponse{
        UID:       review.Request.UID,
        Allowed:   true,
        Patch:     patchBytes,
        PatchType: &pt,
    }

    review.Response = &response
    respBytes, _ := json.Marshal(review)
    w.Header().Set("Content-Type", "application/json")
    w.Write(respBytes)
}

func main() {
    http.HandleFunc("/mutate", mutateFunc)
    fmt.Println("Starting mutating webhook on :8443")
    if err := http.ListenAndServeTLS(":8443", "/tls/tls.crt", "/tls/tls.key", nil); err != nil {
        fmt.Println("Failed to start server:", err)
    }
}
