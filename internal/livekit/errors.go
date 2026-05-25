package livekit

import "errors"

var (
	ErrRecordingAlreadyActive = errors.New("recording already active")
	ErrNoActiveRecording      = errors.New("no active recording")
)
