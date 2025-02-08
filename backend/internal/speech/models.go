package speech

import (
	"errors"
	"sync"
)

const (
	minValue 			= -100
	maxValue 			= 100
)

var (
	ErrInvalidInput		= errors.New("invalid input")
	ErrServiceFailure 	= errors.New("service failure")
	ErrNetworkFailure 	= errors.New("network failure")
	ErrSynthesis 		= errors.New("synthesis error")
	ErrNoAudio 			= errors.New("no audio data received")
)

type Request struct {
	Text   		string `json:"text" binding:"required,min=1"`
	Voice  		string `json:"voice" binding:"omitempty"`
	Pitch  		string `json:"pitch" binding:"omitempty,min=1"`
	Rate   		string `json:"rate" binding:"omitempty,min=1"`
	Volume 		string `json:"volume" binding:"omitempty,min=1"`
}

type Voice struct {
	Name      	string `json:"Name"`
	ShortName 	string `json:"ShortName"`
	Gender    	string `json:"Gender"`
	Locale    	string `json:"Locale"`
}

type VoiceRegistry struct {
	Voices 		map[string]bool
	Mu			sync.RWMutex
}

var ValidVoices = &VoiceRegistry{
	Voices: make(map[string]bool),
}