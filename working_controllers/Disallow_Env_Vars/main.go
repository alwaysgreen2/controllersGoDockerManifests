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

    dep := appsv1.Deployment{}
    _ = json.Unmarshal(review.Request.Object.Raw, &dep)

    allowed := true
    var problems []string

    for _, c := range dep.Spec.Template.Spec.Containers {
        for _, env := range c.Env {
            problems = append(problems,
                fmt.Sprintf("container %q defines forbidden env var %q", c.Name, env.Name))
        }
        if len(c.EnvFrom) > 0 {
            problems = append(problems,
                fmt.Sprintf("container %q uses forbidden envFrom (ConfigMap/Secret import)", c.Name))
        }
    }

    if len(problems) > 0 {
        allowed = false
    }

    response := admissionv1.AdmissionReview{
        TypeMeta: review.TypeMeta,
        Response: &admissionv1.AdmissionResponse{
            UID:     review.Request.UID,
            Allowed: allowed,
        },
    }
    if !allowed {
        msg := "Forbidden environment variables detected:\n" +
            strings.Join(problems, "\n") +
            "\n\nUse volume mounts instead of environment variables."
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
