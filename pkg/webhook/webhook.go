package webhook

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Webhook Server parameters
type WhSvrParameters struct {
	Port     int    // webhook server port
	CertFile string // path to the x509 certificate for https
	KeyFile  string // path to the x509 private key matching `CertFile`
}

type WebhookServer struct {
	Clientset *kubernetes.Clientset
	Server    *http.Server
}

type item struct {
	Spec Spec `json:"spec"`
}

type Spec struct {
	Containers []map[string]string `json:"containers"`
}

//patchStringValue specifies a patch operation for a string.
type PatchStringValue struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

// Serve method for webhook server
func (whsvr *WebhookServer) Serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	if len(body) == 0 {
		glog.Error("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		glog.Errorf("Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	byte_data, _, _, err := jsonparser.Get(body, "request", "object", "metadata", "annotations", "kubectl.kubernetes.io/last-applied-configuration")
	if err != nil {
		glog.Error("bad item")
		http.Error(w, "bad item", http.StatusBadRequest)
	}

	string_data := string(byte_data)
	string_data = strings.Replace(string_data, "\\n", "", -1)
	string_data = strings.Replace(string_data, "\\", "", -1)

	var item item
	if err := json.Unmarshal([]byte(string_data), &item); err != nil {
		glog.Error("bad item")
		http.Error(w, "bad item", http.StatusBadRequest)
	}

	var names []map[string]string
	for _, data := range item.Spec.Containers {
		names = append(names, data)
	}

	err = whsvr.PatchSideCar(names)
	if err != nil {
		glog.Error("patch error")
	}
}

func (whsvr *WebhookServer) PatchSideCar(names []map[string]string) error {
	var err error
	var namespace string
	payload := []PatchStringValue{
		{
			Op:   "add",
			Path: "/spec/template/spec/containers/-",
			Value: v1.Container{
				Name:  "sidecar",
				Image: "theiaide/theia",
				VolumeMounts: []v1.VolumeMount{
					{
						Name:      "shared-data",
						MountPath: "/pod-data",
					},
				},
				Command: []string{"/bin/sh"},
				Args:    []string{"-c", "echo Hello from the debian container > /pod-data/index.html"},
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)

	for _, data := range names {
		if data["namespace"] != "" {
			namespace = data["namespace"]
		} else {
			namespace = "defalut"
		}

		if _, err = whsvr.Clientset.AppsV1().Deployments(namespace).Get(data["name"], meta_v1.GetOptions{}); err != nil {
			return err
		} else {
			if _, err = whsvr.Clientset.AppsV1().Deployments(namespace).Patch(data["name"], types.JSONPatchType, payloadBytes); err != nil {
				return err
			}
		}
	}
	return nil
}
