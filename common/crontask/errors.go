package crontask

import "errors"

var (
	ErrNotFound  = errors.New("[crontask] task not found")
	ErrDuplicate = errors.New("[crontask] task code already exists")
)
