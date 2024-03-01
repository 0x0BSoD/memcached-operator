package controller

import (
	"fmt"
	"os"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"

	cachev1 "github.com/0x0BSoD/memcached-operator/api/v1"
)

func (r *MemcachedReconciler) doFinalizerOperationsForMemcached(cr *cachev1.Memcached) {
	r.Recorder.Event(cr, "Warning", "Deleting",
		fmt.Sprintf("Custom Resource %s is being deleted from the namespace %s",
			cr.Name,
			cr.Namespace))
}

func labelsForMemcached(name, image string) map[string]string {
	return map[string]string{"app.kubernetes.io/name": "Memcached",
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/version":    strings.Split(image, ":")[1],
		"app.kubernetes.io/part-of":    "memcached-operator",
		"app.kubernetes.io/created-by": "controller-manager",
	}
}

func imageForMemcached(memcachedImage cachev1.MemcachedImage) (string, error) {
	var (
		found       bool
		image       string
		imageEnvVar = "MEMCACHED_IMAGE"
	)
	if memcachedImage.Name == "" && memcachedImage.Tag == "" {
		image, found = os.LookupEnv(imageEnvVar)
	} else {
		if memcachedImage.Tag == "" {
			image = fmt.Sprintf("%s:latest", memcachedImage.Name)
		} else {
			image = fmt.Sprintf("%s:%s", memcachedImage.Name, memcachedImage.Tag)
		}
		found = true
	}
	if !found {
		return "", fmt.Errorf("Unable to find %s environment variable or parameter image.name not set", imageEnvVar)
	}
	return image, nil
}

func buildCommand(verboseLevel cachev1.VerboseLevel, memLimitKb int64) []string {
	cmd := []string{"memcached", "-o", "modern"}

	memLimit := (memLimitKb / 1024 / 1024) - 128
	cmd = append(cmd, fmt.Sprintf("--memory-limit=%v", memLimit))

	switch verboseLevel {
	case cachev1.Enable:
		cmd = append(cmd, "-v")
	case cachev1.Moar:
		cmd = append(cmd, "-vv")
	case cachev1.Extreme:
		cmd = append(cmd, "-vvv")
	case cachev1.Disable:
		fmt.Print("Logging disabled")
	}

	return cmd
}

func (r *MemcachedReconciler) deploymentForMemcached(
	memcached *cachev1.Memcached) (*appsv1.Deployment, error) {
	replicas := memcached.Spec.Size
	resources, err := generateResourceRequirements(memcached.Spec.Resources, makeDefaultResources(), "memcached")
	if err != nil {
		return nil, err
	}

	image, err := imageForMemcached(memcached.Spec.Image)
	if err != nil {
		return nil, err
	}
	ls := labelsForMemcached(memcached.Name, image)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      memcached.Name,
			Namespace: memcached.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "kubernetes.io/arch",
												Operator: "In",
												Values:   []string{"amd64", "arm64", "ppc64le", "s390x"},
											},
											{
												Key:      "kubernetes.io/os",
												Operator: "In",
												Values:   []string{"linux"},
											},
										},
									},
								},
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &[]bool{true}[0],
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []corev1.Container{{
						Image:           image,
						Name:            "memcached",
						ImagePullPolicy: corev1.PullIfNotPresent,
						SecurityContext: &corev1.SecurityContext{
							RunAsNonRoot:             &[]bool{true}[0],
							RunAsUser:                &[]int64{1001}[0],
							AllowPrivilegeEscalation: &[]bool{false}[0],
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{
									"ALL",
								},
							},
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: memcached.Spec.ContainerPort,
							Name:          "memcached",
						}},
						Command:   buildCommand(memcached.Spec.Verbose, resources.Limits.Memory().Value()),
						Resources: *resources,
					}},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(memcached, dep, r.Scheme); err != nil {
		return nil, err
	}
	return dep, nil
}
