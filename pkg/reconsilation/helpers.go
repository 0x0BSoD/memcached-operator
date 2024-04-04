package reconsilation

import (
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cachev1 "github.com/0x0BSoD/memcached-operator/api/v1"
)

func retrieveMemcaheds(rc *ReconciliationContext, request *reconcile.Request, dc *cachev1.Memcached) error {
	err := rc.Client.Get(
		rc.Ctx,
		request.NamespacedName,
		dc)
	return err
}

func setOperatorProgressStatus(rc *ReconciliationContext, newState cachev1.ProgressState) error {
	rc.ReqLogger.Info("[reconcile] setOperatorProgressStatus")
	currentState := rc.Memcached.Status.OperatorProgress
	if currentState == newState {
		// early return, no need to ping k8s
		return nil
	}

	patch := client.MergeFrom(rc.Memcached.DeepCopy())
	rc.Memcached.Status.OperatorProgress = newState
	if newState == cachev1.ProgressReady {
		rc.Memcached.Status.ObservedGeneration = rc.Memcached.Generation
	}
	if err := rc.Client.Status().Patch(rc.Ctx, rc.Memcached, patch); err != nil {
		rc.ReqLogger.Error(err, "error updating the Memcached Operator Progress state")
		return err
	}

	return nil
}

func (rc *ReconciliationContext) addFinalizer() error {
	if _, found := rc.Memcached.Annotations[cachev1.NoFinalizerAnnotation]; found {
		return nil
	}

	if !controllerutil.ContainsFinalizer(rc.Memcached, cachev1.Finalizer) && rc.Memcached.GetDeletionTimestamp() == nil {
		rc.ReqLogger.Info("Adding Finalizer for the Memcached")
		controllerutil.AddFinalizer(rc.Memcached, cachev1.Finalizer)

		err := rc.Client.Update(rc.Ctx, rc.Memcached)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rc *ReconciliationContext) listPods(selector map[string]string) (*corev1.PodList, error) {
	rc.ReqLogger.Info("[reconcile] listPods")

	listOptions := &client.ListOptions{
		Namespace:     rc.Memcached.Namespace,
		LabelSelector: labels.SelectorFromSet(selector),
	}

	podList := &corev1.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
	}

	return podList, rc.Client.List(rc.Ctx, podList, listOptions)
}

func PodPtrsFromPodList(podList *corev1.PodList) []*corev1.Pod {
	var pods []*corev1.Pod
	for idx := range podList.Items {
		pod := &podList.Items[idx]
		pods = append(pods, pod)
	}
	return pods
}
