package controller

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cachev1 "github.com/0x0BSoD/memcached-operator/api/v1"
	"github.com/0x0BSoD/memcached-operator/pkg/reconsilation"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
)

// MemcachedReconciler reconciles a Memcached object
type MemcachedReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

var (
	cooldownPeriod     = 20 * time.Second
	minimumRequeueTime = 500 * time.Millisecond
)

// +kubebuilder:rbac:groups=cache.bsod.io,resources=memcacheds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.bsod.io,resources=memcacheds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cache.bsod.io,resources=memcacheds/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

func (r *MemcachedReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	startReconcile := time.Now()

	logger := r.Log.
		WithValues("memcached", req.NamespacedName).
		WithValues("requestNamespace", req.Namespace).
		WithValues("requestName", req.Name).
		// loopID is used to tie all events together that are spawned by the same reconciliation loop
		WithValues("loopID", uuid.New().String())

	log.IntoContext(ctx, logger)

	defer func() {
		reconcileDuration := time.Since(startReconcile).Seconds()
		logger.Info("Reconcile loop completed",
			"duration", reconcileDuration)
	}()

	logger.Info("======== handler::Reconcile has been called")

	rc, err := reconsilation.CreateReconciliationContext(ctx, &req, r.Client, r.Recorder, r.Scheme)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Memcached resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}

		// Error reading the object
		logger.Error(err, "Failed to get Memcached.")
		return ctrl.Result{}, err
	}

	res, err := rc.CalculateReconciliationActions()
	if err != nil {
		logger.Error(err, "calculateReconciliationActions returned an error")
		rc.Recorder.Eventf(rc.Memcached, "Warning", "ReconcileFailed", err.Error())
	}

	if res.Requeue {
		if res.RequeueAfter < minimumRequeueTime {
			res.RequeueAfter = minimumRequeueTime
		}
	}

	return res, err
}

func hasLabel(labels map[string]string, lablel string) bool {
	v, ok := labels[lablel]
	return ok && v == lablel
}

func (r *MemcachedReconciler) SetupWithManager(mgr ctrl.Manager) error {

	memcachedPredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return hasLabel(e.Object.GetLabels(), "app.kubernetes.io/managed-by")
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return hasLabel(e.Object.GetLabels(), "app.kubernetes.io/managed-by")
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return hasLabel(e.ObjectOld.GetLabels(), "app.kubernetes.io/managed-by") || hasLabel(e.ObjectNew.GetLabels(), "app.kubernetes.io/managed-by")
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return hasLabel(e.Object.GetLabels(), "app.kubernetes.io/managed-by")
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1.Memcached{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&appsv1.Deployment{}, builder.WithPredicates(memcachedPredicate)).
		Complete(r)
}

var _ reconcile.Reconciler = &MemcachedReconciler{}
