package controllertest

import (
	"testing"

	"go.uber.org/mock/gomock"
)

// NewGomockController creates a gomock controller tied to t.
func NewGomockController(t *testing.T) *gomock.Controller {
	t.Helper()
	return gomock.NewController(t)
}
