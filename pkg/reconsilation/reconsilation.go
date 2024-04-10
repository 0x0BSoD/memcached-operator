package reconsilation

import (
	"context"

	"github.com/go-logr/logr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cachev1 "github.com/0x0BSoD/memcached-operator/api/v1"
	"github.com/0x0BSoD/memcached-operator/pkg/events"
)

// ReconciliationContext contains all of the input necessary to calculate a list of ReconciliationActions
type ReconciliationContext struct {
	Request   *reconcile.Request
	Client    runtimeClient.Client
	Scheme    *runtime.Scheme
	ReqLogger logr.Logger
	Recorder  record.EventRecorder
	Memcached *cachev1.Memcached
	// According to golang recommendations the context should not be stored in a struct but given that
	// this is passed around as a parameter we feel that its a fair compromise. For further discussion
	// see: golang/go#22602
	Ctx context.Context

	memcachedPods       []*corev1.Pod
	memcachedDeployment *appsv1.Deployment
}

func CreateReconciliationContext(
	ctx context.Context,
	req *reconcile.Request,
	cli runtimeClient.Client,
	rec record.EventRecorder,
	scheme *runtime.Scheme) (*ReconciliationContext, error) {

	reqLogger := log.FromContext(ctx)
	rc := &ReconciliationContext{}
	rc.Request = req
	rc.Client = cli
	rc.Scheme = scheme
	rc.Recorder = &events.LoggingEventRecorder{EventRecorder: rec, ReqLogger: reqLogger}
	rc.ReqLogger = reqLogger
	rc.Ctx = ctx

	rc.ReqLogger = rc.ReqLogger.
		WithValues("namespace", req.Namespace)

	rc.ReqLogger.Info("[handler] CreateReconciliationContext")

	m := &cachev1.Memcached{}
	if err := retrieveMemcaheds(rc, req, m); err != nil {
		// rc.ReqLogger.Error(err, "error in retrieveMemcaheds")
		return nil, err
	}
	rc.Memcached = m

	rc.ReqLogger = rc.ReqLogger.
		WithValues("memcachedName", m.Name)
	log.IntoContext(ctx, rc.ReqLogger)

	return rc, nil
}
