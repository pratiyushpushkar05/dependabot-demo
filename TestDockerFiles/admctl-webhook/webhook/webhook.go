package hook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jinzhu/copier"
	log "github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// TolerationInjector annotates Pods
type TolerationInjector struct {
	Name    string
	Client  client.Client
	Decoder admission.Decoder
	Config  *Config
}

type TolerationSpec struct {
	Spec      corev1.Toleration `yaml:"spec"`
	Namespace string            `yaml:"namespace,omitempty"`
}

type Config struct {
	TolerationSpecs []TolerationSpec `yaml:"tolerationSpecs"`
}

func getTolerationKey(tol corev1.Toleration) string {
	return fmt.Sprintf("%s%s%v%s", tol.Key, tol.Value, tol.Operator, tol.Effect)
}

// Note: We need this, as the kubeclient fails to update the nodes taints if duplicate taints are present.
// If we don't prune the duplicates, the webhook will fail to update the node taints and no pods will be scheduled.
func pruneDuplicates(tolerations []corev1.Toleration) []corev1.Toleration {
	keys := sets.NewString()

	list := []corev1.Toleration{}

	for _, entry := range tolerations {
		tolKey := getTolerationKey(entry)
		if !keys.Has(tolKey) {
			keys.Insert(tolKey)
			list = append(list, entry)
		}
	}

	return list
}

// TolerationInjector adds tolerations to pods.
func (pi *TolerationInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
	if req.Operation != admissionv1.Create {
		return admission.Allowed("Only create operations are handled by this webhook")
	}

	if len(pi.Config.TolerationSpecs) == 0 {
		return admission.Allowed("No tolerations found, skipping")
	}

	pod := &corev1.Pod{}

	err := pi.Decoder.Decode(req, pod)
	if err != nil {
		log.Errorf("Failed to decode pod from request: %v", err)
		return admission.Errored(http.StatusBadRequest, err)
	}

	log.Infof("Injecting tolerations to pod name: %s, namespace: %s", pod.Name, pod.Namespace)

	// deep copy
	config := &Config{}
	_ = copier.CopyWithOption(&config, &pi.Config, copier.Option{IgnoreEmpty: true, DeepCopy: true})

	tolerationsToApply := []corev1.Toleration{}

	for _, tolSpec := range pi.Config.TolerationSpecs {
		if tolSpec.Namespace == "" || tolSpec.Namespace == pod.Namespace {
			tolerationsToApply = append(tolerationsToApply, tolSpec.Spec)
		}
	}

	pod.Spec.Tolerations = append(pod.Spec.Tolerations, pruneDuplicates(tolerationsToApply)...)

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		log.Errorf("Failed to marshall pod: %v", err)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}
