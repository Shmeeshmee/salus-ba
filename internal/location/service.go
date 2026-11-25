package location

import (
	"context"
	"errors"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Ports are what the Service consumes.
type Ports struct {
	Repo  Repository
	Clock clock.Clock
}

// Service is the locations' business rules.
type Service struct {
	p Ports
}

// NewService builds a Service on p.
func NewService(p Ports) *Service {
	return &Service{p: p}
}

// Locations returns the staff's clinic's locations (LC1).
func (s *Service) Locations(_ context.Context, _ Staff) ([]Location, error) {
	return nil, errors.ErrUnsupported
}

// SaveLocation adds a location to the admin's clinic, or changes one at its version (LC1).
func (s *Service) SaveLocation(_ context.Context, _ Staff, _ Location) (Location, error) {
	return Location{}, errors.ErrUnsupported
}

// PublicLocations returns the active locations of a listed clinic, for the patient booking in person (LC1).
func (s *Service) PublicLocations(_ context.Context, _ string) ([]Location, error) {
	return nil, errors.ErrUnsupported
}

// Rooms returns the rooms of a location of the staff's clinic (RM1).
func (s *Service) Rooms(_ context.Context, _ Staff, _ string) ([]Room, error) {
	return nil, errors.ErrUnsupported
}

// SaveRoom adds a room to a location of the admin's clinic, or changes one at its version (RM1).
func (s *Service) SaveRoom(_ context.Context, _ Staff, _ Room) (Room, error) {
	return Room{}, errors.ErrUnsupported
}

// Devices returns the devices of a location of the staff's clinic (RM1).
func (s *Service) Devices(_ context.Context, _ Staff, _ string) ([]Device, error) {
	return nil, errors.ErrUnsupported
}

// SaveDevice adds a device to a location of the admin's clinic, or changes one at its version (RM1).
func (s *Service) SaveDevice(_ context.Context, _ Staff, _ Device) (Device, error) {
	return Device{}, errors.ErrUnsupported
}
