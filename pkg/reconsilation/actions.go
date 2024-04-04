package reconsilation

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cachev1 "github.com/0x0BSoD/memcached-operator/api/v1"
	"github.com/0x0BSoD/memcached-operator/pkg/events"
)

func (rc *ReconciliationContext) ProcessReconcile() (reconcile.Result, error) {
	rc.ReqLogger.Info("[reconciliationContext] processReconcile")

	logger := rc.ReqLogger

	podList, err := rc.listPods(map[string]string{
		cachev1.MemcachedLabel: cachev1.MemcachedLabel,
	})
	if err != nil {
		logger.Error(err, "error listing all pods")
	}
	rc.memcachedPods = PodPtrsFromPodList(podList)

	if recResult := rc.CheckMemcachedDeploymentCreation(); recResult.Completed() {
		return recResult.Output()
	}

	if recResult := rc.CheckMemcachedServiceCreation(); recResult.Completed() {
		return recResult.Output()
	}

	if recResult := rc.CheckMemcachedDeploymentScaling(); recResult.Completed() {
		return recResult.Output()
	}

	if err := setOperatorProgressStatus(rc, cachev1.ProgressReady); err != nil {
		return Error(err).Output()
	}

	rc.ReqLogger.Info("All Staff should now be reconciled.")

	return DoneReconsile().Output()
}

func (rc *ReconciliationContext) ProcessDeletion() ReconcileResult {
	if rc.Memcached.GetDeletionTimestamp() == nil {
		return Continue()
	}

	// If finalizer was removed, we will not do our finalizer processes
	if !controllerutil.ContainsFinalizer(rc.Memcached, cachev1.Finalizer) {
		return DoneReconsile()
	}

	if err := setOperatorProgressStatus(rc, cachev1.ProgressUpdating); err != nil {
		return Error(err)
	}

	origSize := rc.Memcached.Spec.Size
	if rc.Memcached.Status.GetConditionStatus(cachev1.MemcacheDecommission) == corev1.ConditionTrue {
		rc.Memcached.Spec.Size = 0
	}

	if rc.Memcached.Status.GetConditionStatus(cachev1.MemcachedScalingDown) == corev1.ConditionTrue {
		// ScalingDown is still happening
		rc.Recorder.Eventf(rc.Memcached, corev1.EventTypeNormal, events.Decommissioning, "Memcached is decommissioning")
		rc.ReqLogger.V(1).Info("Waiting for the decommission to complete first, before deleting")
		return Continue()
	}

	rc.Memcached.SetFinalizers(nil)
	rc.Memcached.Spec.Size = origSize // Has to be set to original size, since 0 isn't allowed for the Update to succeed

	if err := rc.Client.Update(rc.Ctx, rc.Memcached); err != nil {
		return Error(err)
	}

	return DoneReconsile()
}

func (rc *ReconciliationContext) CalculateReconciliationActions() (reconcile.Result, error) {
	rc.ReqLogger.Info("[handler] calculateReconciliationActions")

	if result := rc.ProcessDeletion(); result.Completed() {
		return result.Output()
	}

	if err := rc.addFinalizer(); err != nil {
		return Error(err).Output()
	}

	result, err := rc.ProcessReconcile()

	return result, err
}
