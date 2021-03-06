/*
Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sslcertificatemanager

import (
	"errors"
	"testing"

	api "github.com/GoogleCloudPlatform/gke-managed-certs/pkg/apis/gke.googleapis.com/v1alpha1"
	compute "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/googleapi"

	"github.com/GoogleCloudPlatform/gke-managed-certs/pkg/client/event"
	"github.com/GoogleCloudPlatform/gke-managed-certs/pkg/client/ssl"
)

type fakeSsl struct {
	err            error
	exists         bool
	sslCertificate *compute.SslCertificate
}

var _ ssl.Ssl = (*fakeSsl)(nil)

func (f fakeSsl) Create(name string, domains []string) error {
	return f.err
}

func (f fakeSsl) Delete(name string) error {
	return f.err
}

func (f fakeSsl) Exists(name string) (bool, error) {
	return f.exists, f.err
}

func (f fakeSsl) Get(name string) (*compute.SslCertificate, error) {
	return f.sslCertificate, f.err
}

func withErr(err error) fakeSsl {
	return fakeSsl{
		err:            err,
		exists:         false,
		sslCertificate: nil,
	}
}

func withExists(err error, exists bool) fakeSsl {
	return fakeSsl{
		err:            err,
		exists:         exists,
		sslCertificate: nil,
	}
}

func withCert(err error, sslCertificate *compute.SslCertificate) fakeSsl {
	return fakeSsl{
		err:            err,
		exists:         false,
		sslCertificate: sslCertificate,
	}
}

type fakeEvent struct {
	backendErrorCnt int
	createCnt       int
	deleteCnt       int
	tooManyCnt      int
}

var _ event.Event = (*fakeEvent)(nil)

func (f *fakeEvent) BackendError(mcrt api.ManagedCertificate, err error) {
	f.backendErrorCnt++
}

func (f *fakeEvent) Create(mcrt api.ManagedCertificate, sslCertificateName string) {
	f.createCnt++
}

func (f *fakeEvent) Delete(mcrt api.ManagedCertificate, sslCertificateName string) {
	f.deleteCnt++
}

func (f *fakeEvent) TooManyCertificates(mcrt api.ManagedCertificate, err error) {
	f.tooManyCnt++
}

var normal = errors.New("normal error")
var quotaExceeded = &googleapi.Error{
	Code: 403,
	Errors: []googleapi.ErrorItem{
		googleapi.ErrorItem{
			Reason: "quotaExceeded",
		},
	},
}
var notFound = &googleapi.Error{
	Code: 404,
}
var cert = &compute.SslCertificate{}
var mcrt = &api.ManagedCertificate{}

func TestCreate(t *testing.T) {
	testCases := []struct {
		sslIn                 ssl.Ssl
		mcrtIn                api.ManagedCertificate
		errOut                error
		tooManyCertsGenerated bool
		backendErrorGenerated bool
		createGenerated       bool
	}{
		{withErr(nil), *mcrt, nil, false, false, true},
		{withErr(quotaExceeded), *mcrt, quotaExceeded, true, false, false},
		{withErr(normal), *mcrt, normal, false, true, false},
	}

	for _, testCase := range testCases {
		event := &fakeEvent{0, 0, 0, 0}
		sut := SslCertificateManager{
			event: event,
			ssl:   testCase.sslIn,
		}

		err := sut.Create("", testCase.mcrtIn)

		if err != testCase.errOut {
			t.Errorf("err %#v, want %#v", err, testCase.errOut)
		}

		if (testCase.tooManyCertsGenerated && event.tooManyCnt != 1) || (!testCase.tooManyCertsGenerated && event.tooManyCnt != 0) {
			t.Errorf("TooManyCertificates events generated: %d, want event to be generated: %t", event.tooManyCnt, testCase.tooManyCertsGenerated)
		}

		if (testCase.backendErrorGenerated && event.backendErrorCnt != 1) || (!testCase.backendErrorGenerated && event.backendErrorCnt != 0) {
			t.Errorf("BackendError events generated: %d, want event to be generated: %t", event.backendErrorCnt, testCase.backendErrorGenerated)
		}

		if (testCase.createGenerated && event.createCnt != 1) || (!testCase.createGenerated && event.createCnt != 0) {
			t.Errorf("Create events generated: %d, want event to be generated: %t", event.createCnt, testCase.createGenerated)
		}
	}
}

func TestDelete(t *testing.T) {
	testCases := []struct {
		sslIn                 ssl.Ssl
		mcrtIn                *api.ManagedCertificate
		errOut                error
		backendErrorGenerated bool
		deleteGenerated       bool
	}{
		{withErr(nil), nil, nil, false, false},
		{withErr(nil), mcrt, nil, false, true},
		{withErr(normal), nil, normal, false, false},
		{withErr(normal), mcrt, normal, true, false},
		{withErr(notFound), nil, nil, false, false},
		{withErr(notFound), mcrt, nil, false, false},
	}

	for _, testCase := range testCases {
		event := &fakeEvent{0, 0, 0, 0}
		sut := SslCertificateManager{
			event: event,
			ssl:   testCase.sslIn,
		}

		err := sut.Delete("", testCase.mcrtIn)

		if err != testCase.errOut {
			t.Errorf("err %#v, want %#v", err, testCase.errOut)
		}

		if (testCase.backendErrorGenerated && event.backendErrorCnt != 1) || (!testCase.backendErrorGenerated && event.backendErrorCnt != 0) {
			t.Errorf("BackendError events generated: %d, want event to be generated: %t", event.backendErrorCnt, testCase.backendErrorGenerated)
		}

		if (testCase.deleteGenerated && event.deleteCnt != 1) || (!testCase.deleteGenerated && event.deleteCnt != 0) {
			t.Errorf("Delete events generated: %d, want event to be generated: %t", event.deleteCnt, testCase.deleteGenerated)
		}
	}
}

func TestExists(t *testing.T) {
	testCases := []struct {
		sslIn          ssl.Ssl
		mcrtIn         *api.ManagedCertificate
		existsOut      bool
		errOut         error
		eventGenerated bool
	}{
		{withExists(nil, false), nil, false, nil, false},
		{withExists(nil, false), mcrt, false, nil, false},
		{withExists(nil, true), nil, true, nil, false},
		{withExists(nil, true), mcrt, true, nil, false},
		{withExists(normal, false), nil, false, normal, false},
		{withExists(normal, false), mcrt, false, normal, true},
		{withExists(normal, true), nil, false, normal, false},
		{withExists(normal, true), mcrt, false, normal, true},
	}

	for _, testCase := range testCases {
		event := &fakeEvent{0, 0, 0, 0}
		sut := SslCertificateManager{
			event: event,
			ssl:   testCase.sslIn,
		}

		exists, err := sut.Exists("", testCase.mcrtIn)

		if err != testCase.errOut {
			t.Errorf("err %#v, want %#v", err, testCase.errOut)
		} else if exists != testCase.existsOut {
			t.Errorf("exists: %t, want %t", exists, testCase.existsOut)
		}

		if (testCase.eventGenerated && event.backendErrorCnt != 1) || (!testCase.eventGenerated && event.backendErrorCnt != 0) {
			t.Errorf("Events generated: %d, want event to be generated: %t", event.backendErrorCnt, testCase.eventGenerated)
		}
	}
}

func TestGet(t *testing.T) {
	testCases := []struct {
		sslIn          ssl.Ssl
		mcrtIn         *api.ManagedCertificate
		certOut        *compute.SslCertificate
		errOut         error
		eventGenerated bool
	}{
		{withCert(nil, nil), nil, nil, nil, false},
		{withCert(nil, nil), mcrt, nil, nil, false},
		{withCert(nil, cert), nil, cert, nil, false},
		{withCert(nil, cert), mcrt, cert, nil, false},
		{withCert(normal, nil), nil, nil, normal, false},
		{withCert(normal, nil), mcrt, nil, normal, true},
		{withCert(normal, cert), nil, nil, normal, false},
		{withCert(normal, cert), mcrt, nil, normal, true},
	}

	for _, testCase := range testCases {
		event := &fakeEvent{0, 0, 0, 0}
		sut := SslCertificateManager{
			event: event,
			ssl:   testCase.sslIn,
		}

		sslCert, err := sut.Get("", testCase.mcrtIn)

		if err != testCase.errOut {
			t.Errorf("err %#v, want %#v", err, testCase.errOut)
		} else if sslCert != testCase.certOut {
			t.Errorf("cert: %#v, want %#v", sslCert, testCase.certOut)
		}

		if (testCase.eventGenerated && event.backendErrorCnt != 1) || (!testCase.eventGenerated && event.backendErrorCnt != 0) {
			t.Errorf("Events generated: %d, want event to be generated: %t", event.backendErrorCnt, testCase.eventGenerated)
		}
	}
}
