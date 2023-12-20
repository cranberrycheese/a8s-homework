/*
Copyright 2023.

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

package controllers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	homeworkv1alpha1 "github.com/cranberrycheese/a8s-homework/api/v1alpha1"
)

// DummyReconciler reconciles a Dummy object
type DummyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=homework.interview.com,resources=dummies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=homework.interview.com,resources=dummies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=homework.interview.com,resources=dummies/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;create

// Reconcile is called when there are changes to a Dummy resource or the pod associated with it.
// It does the following:
// - Log the Dummy resource's name, namespace, and message
// - Sets the SpecEcho status to the message
// - Creates an nginx pod associated with the Dummy resource
// - Writes the nginx pod's Phase to the Dummy's PodStatus status
func (r *DummyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	dummy := &homeworkv1alpha1.Dummy{}
	err := r.Get(ctx, req.NamespacedName, dummy)
	if err != nil {
		if errors.IsNotFound(err) {
			// Was deleted
			return ctrl.Result{}, nil
		} else {
			return ctrl.Result{}, err
		}
	}

	// Log Dummy info
	logger.Info("Name", dummy.Name, "Namespace", dummy.Namespace, "Message", dummy.Spec.Message)

	// Manage Dummy pod
	pod := &corev1.Pod{}
	err = r.Get(ctx, types.NamespacedName{Namespace: dummy.Namespace, Name: getDummyPodName(dummy)}, pod)
	if err != nil {
		if errors.IsNotFound(err) {
			// Pod doesn't exist, create it
			pod = nil

			// Set the Dummy resource as owner of the pod.
			// This will cause updates to the Pod to trigger this reconcile function,
			// and Pod deletion when the Dummy is deleted.
			err := ctrl.SetControllerReference(dummy, pod, r.Scheme)
			if err != nil {
				return ctrl.Result{}, err
			}

			err = r.Create(ctx, pod)
			if err != nil {
				return ctrl.Result{}, err
			}
		} else {
			return ctrl.Result{}, err
		}
	}

	// Update the Status
	var statusChanged bool = false
	if dummy.Status.SpecEcho != dummy.Spec.Message {
		dummy.Status.SpecEcho = dummy.Spec.Message
		statusChanged = true
	}

	if dummy.Status.PodStatus != pod.Status.Phase {
		dummy.Status.PodStatus = pod.Status.Phase
		statusChanged = true
	}

	if statusChanged {
		err := r.Status().Update(ctx, dummy)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DummyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&homeworkv1alpha1.Dummy{}).
		Complete(r)
}

// getDummyPodName returns a name for the pod associated with the Dummy resource.
func getDummyPodName(d *homeworkv1alpha1.Dummy) string {
	return d.Name + "-nginx"
}

// newDummyPod returns a new nginx Pod definition for the given Dummy resource.
func newDummyPod(d *homeworkv1alpha1.Dummy) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"app": d.Name,
			},
			Name:      getDummyPodName(d),
			Namespace: d.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:latest",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 80,
						},
					},
				},
			},
		},
	}
}
