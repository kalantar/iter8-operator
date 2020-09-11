package controllers

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	iter8v1alpha1 "github.com/iter8-tools/iter8-operator/api/v1alpha1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func (r *Iter8Reconciler) crdsForIter8(iter8 *iter8v1alpha1.Iter8) error {
	found := &apiextensionsv1beta1.CustomResourceDefinition{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: "experiments.iter8.tools"}, found)
	if err != nil {
		// could not Get
		if errors.IsNotFound(err) {
			err = InstallCRD(r.Client)
			if err != nil {
				ctrl.Log.Error(err, "Failed to create CustomResourceDefinition")
				return err
			}
		} else {
			ctrl.Log.Error(err, "Failed to create CustomResourceDefinition: search failed")
			return err
		}
	} else {
		// // make sure has owner
		// owner := found.GetOwnerReferences()
		// if 0 == len(owner) {
		// 	ctrl.Log.Info("Updating owner on CustomResourceDefinition", "name", found.Name)
		// 	// Set Iter8 instance as the owner and controller
		// 	controllerutil.SetControllerReference(iter8, found, r.scheme)
		// 	err = r.client.Update(context.TODO(), found)
		// 	return err
		// }
		ctrl.Log.Info("CustomResourceDefinition already present", "name", found.Name)
	}
	return nil
}

var crdMutex sync.Mutex // ensure two workers don't deploy CRDs at same time

// InstallCRD makes sure the CRD has been installed
// CRD is installed from config/iter8/iter8.tools_experiments.yaml
func InstallCRD(cl client.Client) error {
	crdMutex.Lock()
	defer crdMutex.Unlock()

	file, err := os.Open("config/iter8/iter8.tools_experiments.yaml")
	if err != nil {
		return err
	}
	defer file.Close()

	buf := &bytes.Buffer{}
	if _, err = buf.ReadFrom(file); err != nil {
		return err
	}

	crd, err := decodeCRD(string(buf.Bytes()))
	if err != nil {
		return err
	}

	if err = createCRD(cl, crd); err != nil {
		return err
	}

	return nil
}

func decodeCRD(raw string) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	rawJSON, err := yaml.YAMLToJSON([]byte(raw))
	if err != nil {
		ctrl.Log.Error(err, "unable to convert raw data to JSON")
		return nil, err
	}
	obj := &apiextensionsv1beta1.CustomResourceDefinition{}
	if _, _, err = unstructured.UnstructuredJSONScheme.Decode(rawJSON, nil, obj); err != nil {
		ctrl.Log.Error(err, "unable to decode object into Unstructured")
		return nil, err
	}
	if obj.GroupVersionKind().GroupKind().String() == "CustomResourceDefinition.apiextensions.k8s.io" {
		return obj, nil
	}
	return nil, nil

}

func createCRD(cl client.Client, crd *apiextensionsv1beta1.CustomResourceDefinition) error {
	err := cl.Create(context.TODO(), crd)
	if IsTypeObjectProblemInCRDSchemas(err) {
		err = RemoveTypeObjectFieldsFromCRDSchema(context.TODO(), crd)
		if err != nil {
			return err
		}
		err = cl.Create(context.TODO(), crd)
	}
	if err != nil {
		ctrl.Log.Error(err, "error creating CRD")
		return err
	}
	return nil
}

// RemoveTypeObjectFieldsFromCRDSchema works around the problem where OpenShift 3.11 doesn't like "type: object"
// in CRD OpenAPI schemas. This function removes all occurrences from the schema.
func RemoveTypeObjectFieldsFromCRDSchema(ctx context.Context, crd *apiextensionsv1beta1.CustomResourceDefinition) error {
	ctrl.Log.Info("The API server rejected the CRD. Removing type:object fields from the CRD schema and trying again.")

	if crd.Spec.Validation == nil || crd.Spec.Validation.OpenAPIV3Schema == nil {
		return fmt.Errorf("Could not remove type:object fields from CRD schema as no spec.validation.openAPIV3Schema exists")
	}
	removeTypeObjectField(crd.Spec.Validation.OpenAPIV3Schema)
	return nil
}

// IsTypeObjectProblemInCRDSchemas returns true if the error provided is the error usually
// returned by the API server when it doesn't like "type:object" fields in the CRD's OpenAPI Schema.
func IsTypeObjectProblemInCRDSchemas(err error) bool {
	return err != nil && strings.Contains(err.Error(), "must only have \"properties\", \"required\" or \"description\" at the root if the status subresource is enabled")
}

func removeTypeObjectField(schema *apiextensionsv1beta1.JSONSchemaProps) {
	if schema == nil {
		return
	}

	if schema.Type == "object" {
		schema.Type = ""
	}

	removeTypeObjectFieldFromArray(schema.OneOf)
	removeTypeObjectFieldFromArray(schema.AnyOf)
	removeTypeObjectFieldFromArray(schema.AllOf)
	removeTypeObjectFieldFromMap(schema.Properties)
	removeTypeObjectFieldFromMap(schema.PatternProperties)
	removeTypeObjectFieldFromMap(schema.Definitions)
	removeTypeObjectField(schema.Not)

	if schema.Items != nil {
		removeTypeObjectField(schema.Items.Schema)
		removeTypeObjectFieldFromArray(schema.Items.JSONSchemas)
	}
	if schema.AdditionalProperties != nil {
		removeTypeObjectField(schema.AdditionalProperties.Schema)
	}
	if schema.AdditionalItems != nil {
		removeTypeObjectField(schema.AdditionalItems.Schema)
	}
	for k, v := range schema.Dependencies {
		removeTypeObjectField(v.Schema)
		schema.Dependencies[k] = v
	}
}

func removeTypeObjectFieldFromArray(array []apiextensionsv1beta1.JSONSchemaProps) {
	for i, child := range array {
		removeTypeObjectField(&child)
		array[i] = child
	}
}

func removeTypeObjectFieldFromMap(m map[string]apiextensionsv1beta1.JSONSchemaProps) {
	for k, v := range m {
		removeTypeObjectField(&v)
		m[k] = v
	}
}
