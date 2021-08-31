package keptn

import corev1 "k8s.io/api/core/v1"

type KeptnAPIToken struct {
	SecretRef *corev1.ObjectReference `json:"secretRef,omitempty"`
}
