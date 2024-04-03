package reconsilation

import (
	"fmt"
	"os"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cachev1 "github.com/0x0BSoD/memcached-operator/api/v1"
)

func imageForMemcached(memcachedImage cachev1.DockerImage) (string, error) {
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

func labelsForMemcached(name, image string) map[string]string {
	return map[string]string{"app.kubernetes.io/name": "Memcached",
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/version":    strings.Split(image, ":")[1],
		"app.kubernetes.io/part-of":    "memcached-operator",
		"app.kubernetes.io/created-by": "controller-manager",
	}
}

func (rc *ReconciliationContext) deploymentForMemcached() (*appsv1.Deployment, error) {
	rc.ReqLogger.Info("[reconcile_memcached] deploymentForMemcached")

	image, err := imageForMemcached(rc.Memcached.Spec.Image)
	if err != nil {
		return nil, err
	}
	ls := labelsForMemcached(rc.Memcached.Name, image)
	replicas := rc.Memcached.Spec.Size

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rc.Memcached.Name,
			Namespace: rc.Memcached.Namespace,
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
							ContainerPort: rc.Memcached.Spec.ContainerPort,
							Name:          "memcached",
						}},
						Command: []string{
							"memcached",
							"--memory-limit=2048",
							"--max-item-size=32m",
							"modern",
							"--verbose",
							"--conn-limit=1024",
						},
						Resources: rc.Memcached.Spec.Resources,
					}},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(rc.Memcached, dep, rc.Scheme); err != nil {
		return nil, err
	}

	return dep, nil
}

func (rc *ReconciliationContext) CheckMemcachedDeploymentScaling() ReconcileResult {
	logger := rc.ReqLogger
	logger.Info("[reconcile_racks] CheckMemcachedDeploymentScaling")
	m := rc.Memcached
	dep := rc.memcachedDeployments[fmt.Sprintf("%s-%s", rc.Memcached.Name, rc.Memcached.Namespace)]

	if dep == nil {
		return Continue()
	}

	desiredReplicas := rc.Memcached.Spec.Size
	currentReplicas := *dep.Spec.Replicas

	if currentReplicas < desiredReplicas {
		mPatch := client.MergeFrom(m.DeepCopy())

		err := rc.Client.Status().Patch(rc.Ctx, m, mPatch)
		if err != nil {
			logger.Error(err, "error patching memcached for scaling")
			return Error(err)
		}

		rc.ReqLogger.Info(
			"Need to update the memcached's replicas",
			"Memcached", m.Name,
			"currentReplicas", currentReplicas,
			"desiredReplicas", desiredReplicas,
		)

		rc.Recorder.Eventf(rc.Memcached, corev1.EventTypeNormal, ScalingUp,
			"Scaling up %s", m.Name)

		if err := setOperatorProgressStatus(rc, cachev1.ProgressUpdating); err != nil {
			return Error(err)
		}

		patch := client.MergeFrom(dep.DeepCopy())
		dep.Spec.Replicas = &desiredReplicas
		if err := rc.Client.Patch(rc.Ctx, dep, patch); err != nil {
			return Error(err)
		}
	}

	return Continue()
}

func (rc *ReconciliationContext) CheckMemcachedDeploymentCreation() ReconcileResult {
	rc.ReqLogger.Info("[reconcile_memcached] CheckMemcachedDeploymentCreation")

	// Check if the desired Deployment already exists
	currentDeployment := &appsv1.Deployment{}
	err := rc.Client.Get(rc.Ctx,
		types.NamespacedName{
			Name:      rc.Request.Name,
			Namespace: rc.Memcached.Namespace,
		}, currentDeployment)

	if errors.IsNotFound(err) {
		rc.ReqLogger.Info(
			"Creating a new Deployment for",
			"Memcached", rc.Memcached.Name)

		if err := setOperatorProgressStatus(rc, cachev1.ProgressUpdating); err != nil {
			return Error(err)
		}

		dep, err := rc.deploymentForMemcached()
		if err != nil {
			return Error(err)
		}

		if err := rc.Client.Create(rc.Ctx, dep); err != nil {
			return Error(err)
		}
		rc.memcachedDeployments[fmt.Sprintf("%s-%s", rc.Memcached.Name, rc.Memcached.Namespace)] = dep

		rc.Recorder.Eventf(rc.Memcached, corev1.EventTypeNormal, CreatedResource,
			"Created Deployment %s", dep.Name)
		return Continue()
	} else if err != nil {
		rc.ReqLogger.Error(
			err,
			"Could not locate Deployment for",
			"Memcached", rc.Memcached.Name)
		return Error(err)
	}

	return Continue()
}