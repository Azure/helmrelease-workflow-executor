package packager

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Packager interface {
	Package(resources ...v1.Object) (string, error)
}
