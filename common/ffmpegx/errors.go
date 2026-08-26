package ffmpegx

import "errors"

// ErrProcessExists is returned when Start is called with an ID that is already
// managed. Callers must Stop the existing process before starting a replacement.
var ErrProcessExists = errors.New("ffmpeg process already exists")
