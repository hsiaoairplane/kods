/*
Copyright 2024 hsiaoairplane.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"context"
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func SetupStatefulSetWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&appsv1.StatefulSet{}).
		WithDefaulter(&statefulsetDefaulter{mgr.GetClient()}).
		Complete()
}

//+kubebuilder:webhook:path=/mutate--v1-statefulset,mutating=true,failurePolicy=ignore,sideEffects=None,groups="",matchPolicy=Exact,resources=statefulsets,verbs=update,versions=v1,name=statefulset.hsiaoairplane.io,admissionReviewVersions=v1

// statefulsetDefaulter annotates StatefulSets
type statefulsetDefaulter struct {
	client.Client
}

func (a *statefulsetDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	logger := logrus.New()
	sts, ok := obj.(*appsv1.StatefulSet)
	if !ok {
		logger.Errorf("expected a StatefulSet but got a %T", obj)
		return fmt.Errorf("expected a StatefulSet but got a %T", obj)
	}

	// Get the sts last-applied-configuration from annotations
	lastAppliedConfiguration, ok := sts.Annotations["last-applied-configuration"]
	if !ok {
		logger.Info("no last applied configuraiton")
		return nil
	}

	// Unmarshal the last-applied-configuration YAML into an appsv1.StatefulSet object
	lastAppliedSts := &appsv1.StatefulSet{}
	if err := yaml.Unmarshal([]byte(lastAppliedConfiguration), lastAppliedSts); err != nil {
		logger.Errorf("failed to unmarshal last-applied-configuration: %v", err)
		return fmt.Errorf("failed to unmarshal last-applied-configuration: %v", err)
	}

	// Get the last volumeClaimTemplates
	lastAppliedStsVolumeclaimTemplates := lastAppliedSts.Spec.VolumeClaimTemplates
	volumeClaimTemplates := sts.Spec.VolumeClaimTemplates

	if reflect.DeepEqual(lastAppliedStsVolumeclaimTemplates, volumeClaimTemplates) {
		logger.Info("volume claim templates are the same")
		return nil
	}

	// Loop all the volume claim templates and check if the volume claim template size is updated
	for _, volumeClaimTemplate := range volumeClaimTemplates {
		// Get the name of the volume claim template
		volumeClaimTemplateName := volumeClaimTemplate.Name

		// Get the current PVC size
		currentPVCSize := volumeClaimTemplate.Spec.Resources.Requests[corev1.ResourceStorage]

		// Get the PVC size from the last applied configuration
		lastAppliedPVCSize, ok := getLastAppliedPVCSize(lastAppliedSts, volumeClaimTemplateName)
		if !ok {
			// Volume claim template not found in last applied configuration
			logger.Infof("Volume claim template %s not found in last applied configuration", volumeClaimTemplateName)
			continue
		}

		// Compare the PVC sizes
		switch currentPVCSize.Cmp(lastAppliedPVCSize) {
		case 0:
			logger.Infof("Volume claim template %s size matches current spec", volumeClaimTemplateName)
		case -1:
			logger.Warnf("Volume claim template %s size less than current spec", volumeClaimTemplateName)
		case 1:
			logger.Infof("Volume claim template %s size greater than current spec", volumeClaimTemplateName)

			// Orphan delete the Statefulset because the Kubernetes volumeClaimTemplate PVC size is immutable
			// so we nned to orphan delete the StatefulSet and the GitOps will applied the new one
			orphan := metav1.DeletePropagationOrphan
			if err := a.Delete(ctx, sts, &client.DeleteOptions{PropagationPolicy: &orphan}); err != nil {
				logger.Errorf("failed to orphan delete StatefulSet %s/%s: %v", sts.Name, sts.Namespace, err)
				return fmt.Errorf("failed to orphan delete StatefulSet %s/%s: %v", sts.Name, sts.Namespace, err)
			}
		}
	}

	return nil
}

func getLastAppliedPVCSize(lastAppliedSts *appsv1.StatefulSet, volumeClaimTemplateName string) (resource.Quantity, bool) {
	// Iterate through volume claim templates in the last applied StatefulSet configuration
	for _, volumeClaimTemplate := range lastAppliedSts.Spec.VolumeClaimTemplates {
		// Check if the volume claim template name matches
		if volumeClaimTemplate.Name == volumeClaimTemplateName {
			// Found the matching volume claim template
			// Extract the PVC size from its specification
			pvcSize, ok := volumeClaimTemplate.Spec.Resources.Requests[corev1.ResourceStorage]
			if !ok {
				// Return false if PVC size not found
				return resource.Quantity{}, false
			}
			// Return the PVC size and true to indicate success
			return pvcSize, true
		}
	}
	// Return false if the volume claim template with the given name is not found
	return resource.Quantity{}, false
}
