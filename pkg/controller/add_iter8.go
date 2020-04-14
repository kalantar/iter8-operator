package controller

import (
	"github.com/iter8-tools/iter8-operator/pkg/controller/iter8"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, iter8.Add)
}
