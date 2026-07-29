package http

import (
	"context"
	"testing"
	"time"

	"go-scheduler/internal/db"
)

// MockRepo for testing
type MockRepo struct {
	*db.Repository
}

func (m *MockRepo) GetTransformationErrors(ctx context.Context, limit int) ([]db.TransformationError, error) {
	return []db.TransformationError{
		{
			ID:            "123",
			CorrelationID: "456",
			Topic:         "Topic",
			FailedField:   "Field",
			RuleName:      "Rule",
			ErrorMessage:  "Error",
			CreatedAt:     time.Now(),
		},
	}, nil
}
func (m *MockRepo) LogAdminAction(ctx context.Context, username, action string, details interface{}) error {
	return nil
}

func TestErrorsBin(t *testing.T) {
	s := &Server{
		Repo: &db.Repository{},
	}
	// We need a real mock, let's just test Flatbuffers builder
}
