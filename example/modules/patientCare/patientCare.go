package patientCare

type Patient struct {
	ID   int
	Name string
	Age  int
}

func (p *Patient) Create(data ...any) (any, error) {
	// Specific implementation for patients
	return nil, nil
}

func (p *Patient) Read(data ...any) (any, error) {
	// Specific implementation for patients
	return nil, nil
}
