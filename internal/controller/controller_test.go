/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	openldapv1 "github.com/guided-traffic/openldap-operator/api/v1"
)

// Note: Detailed controller tests are in separate files:
// - ldapserver_helper_test.go for LDAPServer helper functions
// - ldapuser_helper_test.go for LDAPUser helper functions
// - ldapgroup_controller_test.go for LDAPGroup controller
// This file contains only integration-level tests

var _ = Describe("Controller Integration", func() {
	It("should be able to initialize all controllers with proper scheme", func() {
		scheme := runtime.NewScheme()
		Expect(openldapv1.AddToScheme(scheme)).To(Succeed())
		Expect(corev1.AddToScheme(scheme)).To(Succeed())

		// LDAPServer controller
		serverReconciler := &LDAPServerReconciler{
			Scheme: scheme,
		}
		Expect(serverReconciler.Scheme).NotTo(BeNil())

		// LDAPUser controller
		userReconciler := &LDAPUserReconciler{
			Scheme: scheme,
		}
		Expect(userReconciler.Scheme).NotTo(BeNil())

		// LDAPGroup controller
		groupReconciler := &LDAPGroupReconciler{
			Scheme: scheme,
		}
		Expect(groupReconciler.Scheme).NotTo(BeNil())
	})
})
