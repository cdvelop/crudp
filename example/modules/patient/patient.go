package patient

import "context"

type Handler struct{}

type Patient struct {
	ID   int
	Name string
	Age  int
}

func (h *Handler) Create(ctx context.Context, data ...any) (any, error) {
	// Specific implementation for patients
	return nil, nil
}

func (h *Handler) Read(ctx context.Context, data ...any) (any, error) {
	// Specific implementation for patients
	return nil, nil
}
