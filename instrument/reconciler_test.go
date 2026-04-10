package instrument

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/u-ctf/controller-fwk/mocks"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestNewInstrumentedReconciler(t *testing.T) {
	ctrlr := gomock.NewController(t)
	defer ctrlr.Finish()

	mockInstrumenter := mocks.NewMockInstrumenter(ctrlr)
	mockReconciler := mocks.NewMockTypedReconciler[reconcile.Request](ctrlr)

	instrumentedReconciler := NewInstrumentedReconciler(mockInstrumenter, mockReconciler)

	if instrumentedReconciler.Instrumenter != mockInstrumenter {
		t.Errorf("expected instrumenter to be set correctly")
	}

	if instrumentedReconciler.internalReconciler != mockReconciler {
		t.Errorf("expected internal reconciler to be set correctly")
	}
}

func TestInstrumentedReconciler_Reconcile_Success(t *testing.T) {
	ctrlr := gomock.NewController(t)
	defer ctrlr.Finish()

	mockInstrumenter := mocks.NewMockInstrumenter(ctrlr)
	mockReconciler := mocks.NewMockTypedReconciler[reconcile.Request](ctrlr)

	instrumentedReconciler := NewInstrumentedReconciler(mockInstrumenter, mockReconciler)

	ctx := context.Background()
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "test-namespace",
			Name:      "test-name",
		},
	}

	expectedResult := reconcile.Result{Requeue: false}
	ctx = context.Background()

	// Set up expectations
	mockInstrumenter.EXPECT().GetContextForRequest(req).Return(&ctx, true)
	mockInstrumenter.EXPECT().NewLogger(gomock.Any()).Return(logr.New(nil))
	mockInstrumenter.EXPECT().StartSpan(gomock.Any(), gomock.Any(), gomock.Any()).Return(ctx, trace.SpanFromContext(ctx))
	mockReconciler.EXPECT().Reconcile(gomock.Any(), req).Return(expectedResult, nil)
	mockInstrumenter.EXPECT().Cleanup(&ctx, req)

	// Test the Reconcile method
	result, err := instrumentedReconciler.Reconcile(ctx, req)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result != expectedResult {
		t.Errorf("expected result %+v, got %+v", expectedResult, result)
	}
}

func TestInstrumentedReconciler_Reconcile_WithError(t *testing.T) {
	ctrlr := gomock.NewController(t)
	defer ctrlr.Finish()

	mockInstrumenter := mocks.NewMockInstrumenter(ctrlr)
	mockReconciler := mocks.NewMockTypedReconciler[reconcile.Request](ctrlr)

	instrumentedReconciler := NewInstrumentedReconciler(mockInstrumenter, mockReconciler)

	ctx := context.Background()
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "test-namespace",
			Name:      "test-name",
		},
	}

	expectedResult := reconcile.Result{Requeue: true}
	expectedError := errors.New("reconcile error")
	ctx = context.Background()

	// Set up expectations
	mockInstrumenter.EXPECT().GetContextForRequest(req).Return(&ctx, true)
	mockInstrumenter.EXPECT().NewLogger(gomock.Any()).Return(logr.New(nil))
	mockInstrumenter.EXPECT().StartSpan(gomock.Any(), gomock.Any(), gomock.Any()).Return(ctx, trace.SpanFromContext(ctx))
	mockReconciler.EXPECT().Reconcile(gomock.Any(), req).Return(expectedResult, expectedError)
	mockInstrumenter.EXPECT().Cleanup(&ctx, req)

	// Test the Reconcile method with error
	result, err := instrumentedReconciler.Reconcile(ctx, req)

	if err != expectedError {
		t.Errorf("expected error %v, got %v", expectedError, err)
	}

	if result != expectedResult {
		t.Errorf("expected result %+v, got %+v", expectedResult, result)
	}
}

func TestInstrumentedReconciler_LoggerContext(t *testing.T) {
	ctrlr := gomock.NewController(t)
	defer ctrlr.Finish()

	mockInstrumenter := mocks.NewMockInstrumenter(ctrlr)
	mockReconciler := mocks.NewMockTypedReconciler[reconcile.Request](ctrlr)

	instrumentedReconciler := NewInstrumentedReconciler(mockInstrumenter, mockReconciler)

	expectedResult := reconcile.Result{Requeue: false}
	ctx := context.Background()
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "test-namespace",
			Name:      "test-name",
		},
	}

	mockInstrumenter.EXPECT().GetContextForRequest(req).Return(&ctx, true)
	mockInstrumenter.EXPECT().NewLogger(gomock.Any()).Return(logr.New(logf.NullLogSink{}))
	mockInstrumenter.EXPECT().StartSpan(gomock.Any(), gomock.Any(), gomock.Any()).Return(ctx, trace.SpanFromContext(ctx))

	mockReconciler.EXPECT().Reconcile(gomock.Any(), req).DoAndReturn(func(localCtx context.Context, req reconcile.Request) (reconcile.Result, error) {
		logger := logf.FromContext(localCtx)

		switch logger.GetSink().(type) {
		default:
			return expectedResult, fmt.Errorf("logger sink is not logf.NullLogSink")
		case logf.NullLogSink:
			return expectedResult, nil
		}
	})
	mockInstrumenter.EXPECT().Cleanup(&ctx, req)

	result, err := instrumentedReconciler.Reconcile(ctx, req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result != expectedResult {
		t.Errorf("expected result %+v, got %+v", expectedResult, result)
	}
}
