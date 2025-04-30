package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"

    admissionv1 "k8s.io/api/admission/v1"
    appsv1      "k8s.io/api/apps/v1"
    corev1      "k8s.io/api/core/v1"
    metav1     "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func admitFunc(w http.ResponseWriter, r *http.Request) {
    var review admissionv1.AdmissionReview
    body, _ := ioutil.ReadAll(r.Body)
    _ = json.Unmarshal(body, &review)

    dep := appsv1.Deployment{}
    _ = json.Unmarshal(review.Request.Object.Raw, &dep)

    allowed := true
    var msg string

    // Validate each container's requests==limits for cpu & memory
    for _, c := range dep.Spec.Template.Spec.Containers {
        reqs := c.Resources.Requests
        lims := c.Resources.Limits

        cpuReq, okCPUReq := reqs[corev1.ResourceCPU]
        cpuLim, okCPULim := lims[corev1.ResourceCPU]
        memReq, okMemReq := reqs[corev1.ResourceMemory]
        memLim, okMemLim := lims[corev1.ResourceMemory]

        if !okCPUReq || !okCPULim || cpuReq.Cmp(cpuLim) != 0 ||
           !okMemReq || !okMemLim || memReq.Cmp(memLim) != 0 {
            allowed = false
            msg = fmt.Sprintf(
                "container %q must have CPU and Memory requests AND limits set and equal", 
                c.Name,
            )
            break
        }
    }

    response := admissionv1.AdmissionReview{
        TypeMeta: review.TypeMeta,
        Response: &admissionv1.AdmissionResponse{
            UID:     review.Request.UID,
            Allowed: allowed,
        },
    }
    if !allowed {
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
