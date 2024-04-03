package reconsilation

import (
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

const (
	// Events
	DeletingStuckPod string = "DeletingStuckPod"
	CreatedResource  string = "CreatedResource"
	ScalingUp        string = "ScalingUp"
	ScalingDown      string = "ScalingDown"
	Decommissioning  string = "Decommissioning"
	Unhealthy        string = "Unhealthy"
)

type LoggingEventRecorder struct {
	record.EventRecorder
	ReqLogger logr.Logger
}

func (r *LoggingEventRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	r.ReqLogger.Info(message, "reason", reason, "eventType", eventtype)
	r.EventRecorder.Event(object, eventtype, reason, message)
}

func (r *LoggingEventRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	r.ReqLogger.Info(fmt.Sprintf(messageFmt, args...), "reason", reason, "eventType", eventtype)
	r.EventRecorder.Eventf(object, eventtype, reason, messageFmt, args...)
}

func (r *LoggingEventRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
	r.ReqLogger.Info(fmt.Sprintf(messageFmt, args...), "reason", reason, "eventType", eventtype)
	r.EventRecorder.AnnotatedEventf(object, annotations, eventtype, reason, messageFmt, args...)
}
