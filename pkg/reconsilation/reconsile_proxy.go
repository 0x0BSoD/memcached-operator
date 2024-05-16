package reconsilation

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cachev1 "github.com/0x0BSoD/memcached-operator/api/v1"
	"github.com/0x0BSoD/memcached-operator/pkg/events"
)

// imageForProxy
// TODO: Repeated code
func imageForProxy(proxyImage cachev1.DockerImage) string {
	var (
		found       bool
		image       string
		imageEnvVar = "PROXY_IMAGE"
	)
	if proxyImage.Name == "" && proxyImage.Tag == "" {
		image, found = os.LookupEnv(imageEnvVar)
	} else {
		if proxyImage.Tag == "" {
			image = fmt.Sprintf("%s:latest", proxyImage.Name)
		} else {
			image = fmt.Sprintf("%s:%s", proxyImage.Name, proxyImage.Tag)
		}
		found = true
	}
	if !found {
		image = cachev1.ProxyDefaultImage
	}
	return image
}

func labelsForProxy(name, image string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "Memcachd-Proxy",
		"app.kubernetes.io/instance":   fmt.Sprintf("%s-proxy", name),
		"app.kubernetes.io/version":    strings.Split(image, ":")[1],
		"app.kubernetes.io/part-of":    "memcached-operator",
		"app.kubernetes.io/created-by": "controller-manager",
	}
}

func (rc *ReconciliationContext) serviceForProxy() (*corev1.Service, error) {
	rc.ReqLogger.Info("[reconcile_proxy] serviceForProxy")

	image := imageForProxy(rc.Memcached.Spec.Proxy.Image)
	ls := labelsForProxy(rc.Memcached.Name, image)

	listenPort, _ := strconv.ParseInt(
		strings.Split(rc.Memcached.Spec.Proxy.Config.Listen, ":")[1],
		10,
		32,
	)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-proxy", rc.Memcached.Name),
			Namespace: rc.Memcached.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "proxy",
					Port: int32(listenPort),
				},
			},
			Selector: ls,
			Type:     corev1.ServiceTypeClusterIP,
		},
	}

	if err := ctrl.SetControllerReference(rc.Memcached, svc, rc.Scheme); err != nil {
		return nil, err
	}

	return svc, nil
}

func (rc *ReconciliationContext) deploymentForProxy() (*appsv1.Deployment, error) {
	rc.ReqLogger.Info("[reconcile_proxy] deploymentForProxy")

	image := imageForProxy(rc.Memcached.Spec.Proxy.Image)
	ls := labelsForProxy(rc.Memcached.Name, image)
	replicas := rc.Memcached.Spec.Proxy.Replicas

	listenPort, _ := strconv.ParseInt(
		strings.Split(rc.Memcached.Spec.Proxy.Config.Listen, ":")[1],
		10,
		32,
	)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-proxy", rc.Memcached.Name),
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
												Values: []string{
													"amd64",
													"arm64",
													"ppc64le",
													"s390x",
												},
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
						Name:            "proxy",
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
							ContainerPort: int32(listenPort),
							Name:          "proxy",
						}},
						Command: []string{
							"nutcracker",
							"-c",
							"/etc/config/twem-config.yaml",
							"-v",
							"7",
						},
						Resources: rc.Memcached.Spec.Proxy.Resources,
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

func (rc *ReconciliationContext) CheckProxyDeploymentScaling() ReconcileResult {
	logger := rc.ReqLogger
	m := rc.Memcached
	dep := rc.memcachedDeployment

	if dep == nil {
		return Continue()
	}

	logger.Info("[reconcile_proxy] CheckProxyDeploymentScaling")

	desiredReplicas := rc.Memcached.Spec.Size
	currentReplicas := *dep.Spec.Replicas

	if currentReplicas != desiredReplicas {
		mPatch := client.MergeFrom(m.DeepCopy())

		err := rc.Client.Status().Patch(rc.Ctx, m, mPatch)
		if err != nil {
			logger.Error(err, "error patching proxy for scaling")
			return Error(err)
		}

		rc.ReqLogger.Info(
			"Need to update the proxy's replicas",
			"Memcached-Proxy", m.Name,
			"currentReplicas", currentReplicas,
			"desiredReplicas", desiredReplicas,
		)

		if currentReplicas > desiredReplicas {
			rc.Recorder.Eventf(rc.Memcached, corev1.EventTypeNormal, events.ScalingDown,
				"Scaling Down %s", m.Name)
		} else if currentReplicas < desiredReplicas {
			rc.Recorder.Eventf(rc.Memcached, corev1.EventTypeNormal, events.ScalingUp,
				"Scaling Up %s", m.Name)
		}

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

func (rc *ReconciliationContext) CheckProxyDeploymentCreation() ReconcileResult {
	rc.ReqLogger.Info("[reconcile_proxy] CheckProxyDeploymentCreation")

	// Check if the desired Deployment already exists
	currentDeployment := &appsv1.Deployment{}
	err := rc.Client.Get(rc.Ctx,
		types.NamespacedName{
			Name:      fmt.Sprintf("%s-proxy", rc.Memcached.Name),
			Namespace: rc.Memcached.Namespace,
		}, currentDeployment)

	if errors.IsNotFound(err) {
		rc.ReqLogger.Info(
			"Creating a new Deployment for",
			"Memcached-Proxy", rc.Memcached.Name)

		if err := setOperatorProgressStatus(rc, cachev1.ProgressUpdating); err != nil {
			return Error(err)
		}

		dep, err := rc.deploymentForProxy()
		if err != nil {
			return Error(err)
		}

		if err := rc.Client.Create(rc.Ctx, dep); err != nil {
			return Error(err)
		}

		rc.proxyDeployment = dep

		rc.Recorder.Eventf(rc.Memcached, corev1.EventTypeNormal, events.CreatedResource,
			"Created Deployment %s", dep.Name)
		return Continue()
	} else if err != nil {
		rc.ReqLogger.Error(
			err,
			"Could not locate Deployment for",
			"Memcached-Proxy", rc.Memcached.Name)
		return Error(err)
	}

	rc.proxyDeployment = currentDeployment

	return Continue()
}

func (rc *ReconciliationContext) CheckProxyServiceCreation() ReconcileResult {
	if rc.proxyDeployment == nil {
		return Continue()
	}

	rc.ReqLogger.Info("[reconcile_proxy] CheckProxyServiceCreation")

	currentService := &corev1.Service{}
	err := rc.Client.Get(rc.Ctx,
		types.NamespacedName{
			Name:      fmt.Sprintf("%s-proxy", rc.Memcached.Name),
			Namespace: rc.Request.Namespace,
		}, currentService)
	if errors.IsNotFound(err) {
		rc.ReqLogger.Info(
			"Creating a new Service for",
			"Memcached-Proxy", rc.Memcached.Name)

		if err := setOperatorProgressStatus(rc, cachev1.ProgressUpdating); err != nil {
			return Error(err)
		}

		svc, err := rc.serviceForProxy()
		if err != nil {
			return Error(err)
		}

		if err := rc.Client.Create(rc.Ctx, svc); err != nil {
			return Error(err)
		}

		rc.Recorder.Eventf(rc.Memcached, corev1.EventTypeNormal, events.CreatedResource,
			"Created Service %s", svc.Name)
		return Continue()
	} else if err != nil {
		rc.ReqLogger.Error(
			err,
			"Could not locate Service for",
			"Memcached-Proxy", rc.Memcached.Name)
		return Error(err)
	}

	return Continue()
}
