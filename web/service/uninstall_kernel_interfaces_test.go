package service

import (
	"errors"
	"reflect"
	"testing"
)

func TestStopKernelCoreInterfacesAttemptsEveryCore(t *testing.T) {
	var called []string
	report := &UninstallReport{}
	steps := []kernelInterfaceStop{
		{name: "wg-c", stop: func() error { called = append(called, "wg-c"); return nil }},
		{name: "awg", stop: func() error { called = append(called, "awg"); return errors.New("test failure") }},
		{name: "gre", stop: func() error { called = append(called, "gre"); return nil }},
	}

	stopKernelCoreInterfacesWith(report, steps)

	if want := []string{"wg-c", "awg", "gre"}; !reflect.DeepEqual(called, want) {
		t.Fatalf("stop order/calls = %v, want %v", called, want)
	}
	if len(report.Errors) != 1 || report.Errors[0] != "stop awg kernel interfaces: test failure" {
		t.Fatalf("uninstall report errors = %v", report.Errors)
	}
}
