//go:build !ignore_uncovered
// +build !ignore_uncovered

// Package v1alpha2 contains API Schema definitions for the acn v1alpha2 API group
// +kubebuilder:object:generate=true
// +groupName=multitenancy.acn.azure.com
package v1alpha2

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "multitenancy.acn.azure.com", Version: "v1alpha2"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
