package model

import "errors"

var (
	ErrNotFound        = errors.New("entity not found")
	ErrDuplicate       = errors.New("duplicate entity")
	ErrInvalidState    = errors.New("invalid state transition")
	ErrMissingStandard = errors.New("internal standard missing")
	ErrStandardLocked  = errors.New("internal standard locked")
	ErrSealedBatch     = errors.New("batch is sealed")
	ErrTempConflict    = errors.New("duplicate temperature in sequence")
	ErrUnitMismatch    = errors.New("frequency unit mismatch (ppm vs hz)")
	ErrNoStandardPeak  = errors.New("no standard peak at temperature")
	ErrInvalidInput    = errors.New("invalid input")
)
