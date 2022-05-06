package fake

import (
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
)

// FakeRecorder is used as a fake during tests
type FakeRecorder struct {
	Log logr.Logger
}

func (f *FakeRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	msg := fmt.Sprintf("%s %s %s", eventtype, reason, message)
	f.Log.Info(msg)
}

func (f *FakeRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	msg := fmt.Sprintf(eventtype+" "+reason+" "+messageFmt, args...)
	f.Log.Info(msg)
}

func (f *FakeRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
	f.Eventf(object, eventtype, reason, messageFmt, args...)
}

func NewFakeRecorder(log logr.Logger) *FakeRecorder {
	return &FakeRecorder{
		Log: log,
	}
}
