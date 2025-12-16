package hook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/jinzhu/copier"
	log "github.com/sirupsen/logrus"
	admv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// CertInitializer annotates Pods
type CertInitializer struct {
	Name         string
	Client       client.Client
	Decoder      admission.Decoder
	Config       *ConfigTypes
	CertInjector bool
}

type ConfigTypes struct {
	Rhel   *Config
	Debian *Config
}

type Config struct {
	Containers     []corev1.Container   `yaml:"containers"`
	InitContainers []corev1.Container   `yaml:"initContainers"`
	Volumes        []corev1.Volume      `yaml:"volumes"`
	VolumeMounts   []corev1.VolumeMount `yaml:"volumeMounts"`
}

func shoudInject(pod *corev1.Pod, ci *CertInitializer, os string) bool {
	var config *Config
	if os == "debian" {
		config = ci.Config.Debian
	} else {
		config = ci.Config.Rhel
	}

	shouldInject := ci.CertInjector
	alreadyInjected := false

	for _, initContainer := range pod.Spec.InitContainers {
		if initContainer.Name == config.InitContainers[0].Name {
			alreadyInjected = true
		}
	}
	if pod.Annotations["cert-initializer/inject"] != "" {
		var err error
		shouldInject, err = strconv.ParseBool(pod.Annotations["cert-initializer/inject"])

		if err != nil {
			shouldInject = false
			log.Info("Should Inject Error", err.Error())
		}
	}

	shouldInject = shouldInject && !alreadyInjected

	log.Info("Should Inject: ", shouldInject)
	return shouldInject
}

func getOS(pod *corev1.Pod) string {
	os := pod.Annotations["cert-initializer/container-os"]
	if os == "" {
		os = "rhel"
	}
	return os
}

// CertInitializer adds an annotation to every incoming pods.
func (ci *CertInitializer) Handle(ctx context.Context, req admission.Request) admission.Response {
	if req.Operation != admv1.Create {
		return admission.Allowed("Only create operations are handled by this webhook")
	}

	var containers []string = []string{}
	pod := &corev1.Pod{}

	err := ci.Decoder.Decode(req, pod)
	if err != nil {
		log.Errorf("Failed to decode pod from request: %v", err)
		return admission.Errored(http.StatusBadRequest, err)
	}

	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	os := getOS(pod)

	for _, container := range pod.Spec.Containers {
		containers = append(containers, container.Name)
	}

	log.Info("===========================================")
	log.Info(fmt.Sprintf("Inject request for POD with containers %s in namespace %s", strings.Join(containers, ","), pod.Namespace))

	shoudInjectCertInitializer := shoudInject(pod, ci, os)

	if shoudInjectCertInitializer {
		log.Info("Injecting cert-initializer")
		// deep copy
		config := &Config{}

		if os == "debian" {
			_ = copier.CopyWithOption(&config, &ci.Config.Debian, copier.Option{IgnoreEmpty: true, DeepCopy: true})
		} else {
			_ = copier.CopyWithOption(&config, &ci.Config.Rhel, copier.Option{IgnoreEmpty: true, DeepCopy: true})
		}

		pod.Spec.InitContainers = append(config.InitContainers, pod.Spec.InitContainers...)
		pod.Spec.Volumes = append(pod.Spec.Volumes, config.Volumes...)

		// Insert volume mounts in all the containers
		for i := 0; i < len(pod.Spec.Containers); i++ {
			pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, config.VolumeMounts...)
		}

		// Insert volume mounts in all the InitContainers
		for i := 0; i < len(pod.Spec.InitContainers); i++ {
			pod.Spec.InitContainers[i].VolumeMounts = append(pod.Spec.InitContainers[i].VolumeMounts, config.VolumeMounts...)
		}

		log.Info("CertInitializer ", ci.Name, " injected.")
	} else {
		log.Info("Inject not needed.")
	}

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		log.Errorf("Failed to marshall pod: %v", err)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	log.Info("===========================================")

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}
