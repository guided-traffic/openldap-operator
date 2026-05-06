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

// Package v1 contains API Schema definitions for the openldap v1 API group
// +kubebuilder:object:generate=true
// +groupName=openldap.guided-traffic.com
package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "openldap.guided-traffic.com", Version: "v1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// Builder registers Kubernetes API types for a GroupVersion without depending
// on controller-runtime, whose scheme.Builder is deprecated for use in api packages.
type Builder struct {
	GroupVersion schema.GroupVersion
	runtime.SchemeBuilder
}

// Register adds the given types to the SchemeBuilder under the configured GroupVersion.
func (b *Builder) Register(types ...runtime.Object) *Builder {
	b.SchemeBuilder.Register(func(scheme *runtime.Scheme) error {
		scheme.AddKnownTypes(b.GroupVersion, types...)
		metav1.AddToGroupVersion(scheme, b.GroupVersion)
		return nil
	})
	return b
}
