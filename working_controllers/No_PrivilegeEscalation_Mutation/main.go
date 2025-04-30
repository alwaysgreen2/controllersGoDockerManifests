package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"

    admissionv1 "k8s.io/api/admission/v1"
    corev1      "k8s.io/api/core/v1"
    appsv1      "k8s.io/api/apps/v1"
)

func mutate(w http.ResponseWriter, r *http.Request) {
    var review admissionv1.AdmissionReview
    body, _ := ioutil.ReadAll(r.Body)
    _ = json.Unmarshal(body, &review)

    var patches []map[string]interface{}

    // Helper to emit patches for a container slice
    handleContainers := func(containers []corev1.Container, basePath string) {
        for i, c := range containers {
            sc := c.SecurityContext
            pathSC := fmt.Sprintf("%s/%d/securityContext", basePath, i)
            pathAP := fmt.Sprintf("%s/%d/securityContext/allowPrivilegeEscalation", basePath, i)

            if sc == nil {
                // container.SecurityContext == nil  → add entire object
                patches = append(patches, map[string]interface{}{
                    "op":    "add",
                    "path":  pathSC,
                    "value": map[string]bool{"allowPrivilegeEscalation": false},
                })
            } else if sc.AllowPrivilegeEscalation == nil || *sc.AllowPrivilegeEscalation {
                // object exists but field missing or true → add/replace field
                patches = append(patches, map[string]interface{}{
                    "op":    "add",
                    "path":  pathAP,
                    "value": false,
                })
            }
        }
    }

    // Dispatch on Pod vs Deployment
    switch review.Request.Kind.Kind {
    case "Pod":
        pod := corev1.Pod{}
        _ = json.Unmarshal(review.Request.Object.Raw, &pod)
        handleContainers(pod.Spec.Containers, "/spec/containers")
        handleContainers(pod.Spec.InitContainers, "/spec/initContainers")

    case "Deployment":
        dep := appsv1.Deployment{}
        _ = json.Unmarshal(review.Request.Object.Raw, &dep)
        // PodTemplate sits under spec.template.spec
        handleContainers(dep.Spec.Template.Spec.Containers, "/spec/template/spec/containers")
        handleContainers(dep.Spec.Template.Spec.InitContainers, "/spec/template/spec/initContainers")
    }

    // Build the AdmissionResponse
    patchBytes, _ := json.Marshal(patches)
    pt := admissionv1.PatchTypeJSONPatch
    resp := admissionv1.AdmissionResponse{
        UID:       review.Request.UID,
        Allowed:   true,
        Patch:     patchBytes,
        PatchType: &pt,
    }
    review.Response = &resp

    respBytes, _ := json.Marshal(review)
    w.Header().Set("Content-Type", "application/json")
    w.Write(respBytes)
}

func main() {
    http.HandleFunc("/mutate", mutate)
    fmt.Println("Starting mutating webhook on :8443")
    if err := http.ListenAndServeTLS(":8443", "/tls/tls.crt", "/tls/tls.key", nil); err != nil {
        fmt.Println("Failed to start server:", err)
    }
}
