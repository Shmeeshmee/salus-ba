package diagnosticreport

import (
	"context"
	"errors"
	"io"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Ports are what the Service consumes.
type Ports struct {
	Repo  Repository
	Clock clock.Clock
}

// Service is the lab results' business rules.
type Service struct {
	p Ports
}

// NewService builds a Service on p.
func NewService(p Ports) *Service {
	return &Service{p: p}
}

// Attach attaches a report to a visit of the staff's clinic, preliminary (LB1).
func (s *Service) Attach(_ context.Context, _ Staff, _ string, _ Draft) (Report, error) {
	return Report{}, errors.ErrUnsupported
}

// Change changes a preliminary report at its version (LB1).
func (s *Service) Change(_ context.Context, _ Staff, _ string, _ Draft) (Report, error) {
	return Report{}, errors.ErrUnsupported
}

// AttachPDF stores the lab's PDF of a preliminary report, replacing one already there (LB1).
func (s *Service) AttachPDF(_ context.Context, _ Staff, _ string, _ io.Reader, _ int64) (Report, error) {
	return Report{}, errors.ErrUnsupported
}

// Release releases a report at its version: the practitioner's review, after which the patient sees it (LB1).
func (s *Service) Release(_ context.Context, _ Staff, _ string, _ int) (Report, error) {
	return Report{}, errors.ErrUnsupported
}

// ForVisit returns a visit's reports to the staff who may see the visit (LB1).
func (s *Service) ForVisit(_ context.Context, _ Staff, _ string) ([]Report, error) {
	return nil, errors.ErrUnsupported
}

// Read returns one report of the staff's clinic (LB1).
func (s *Service) Read(_ context.Context, _ Staff, _ string) (Report, error) {
	return Report{}, errors.ErrUnsupported
}

// PDF opens the PDF of a report of the staff's clinic and returns it with its size (LB1).
func (s *Service) PDF(_ context.Context, _ Staff, _ string) (io.ReadCloser, int64, error) {
	return nil, 0, errors.ErrUnsupported
}

// Mine returns the patient's released reports, newest first (LB1).
func (s *Service) Mine(_ context.Context, _ Patient) ([]Report, error) {
	return nil, errors.ErrUnsupported
}

// MyReport returns one of the patient's released reports (LB1).
func (s *Service) MyReport(_ context.Context, _ Patient, _ string) (Report, error) {
	return Report{}, errors.ErrUnsupported
}

// MyPDF opens the PDF of one of the patient's released reports and returns it with its size (LB1).
func (s *Service) MyPDF(_ context.Context, _ Patient, _ string) (io.ReadCloser, int64, error) {
	return nil, 0, errors.ErrUnsupported
}
