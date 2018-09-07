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

package utils

import (
	"testing"
)

func newRandomName(t *testing.T) string {
	if name, err := RandomName(); err != nil {
		t.Errorf("Failed to create random name: %v", err)
		return ""
	} else {
		return name
	}
}

func TestRandomName_NonEmptyNameShorterThanLimit(t *testing.T) {
	if name := newRandomName(t); len(name) <= 0 || len(name) >= 64 {
		t.Errorf("Random name %s has %d characters, should have between 0 and 63", name, len(name))
	}
}

func TestRandomName_TwiceReturnsDifferent(t *testing.T) {
	if name := newRandomName(t); name == newRandomName(t) {
		t.Errorf("RandomName called twice returned the same name %s", name)
	}
}
