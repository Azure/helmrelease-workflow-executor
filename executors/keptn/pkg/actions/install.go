package actions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Azure/orkestra-workflow-executor/executors/keptn/pkg/keptn"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Install(ctx context.Context, cancel context.CancelFunc, dir string, clientSet client.Client, keptnCli *keptn.Keptn, cm types.NamespacedName, interval time.Duration) error {
	obj := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cm.Name,
			Namespace: cm.Namespace,
		},
	}
	if err := clientSet.Get(ctx, cm, obj); err != nil {
		return err
	}

	if obj.Data == nil {
		return fmt.Errorf("configmap data field cannot be nil")
	}

	if len(obj.Data) == 0 {
		return fmt.Errorf("configmap data field cannot be empty")
	}

	for name, contents := range obj.Data {
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := f.WriteString(contents); err != nil {
			return err
		}
	}

	return nil
}
