package speech

import (
	"fmt"
	"strings"
	"regexp"
	"strconv"
	"encoding/json"
	"time"
	"errors"
)

func BuildTTSConfigMessage() string {
    config := map[string]interface{}{
        "context": map[string]interface{}{
            "synthesis": map[string]interface{}{
                "audio": map[string]interface{}{
                    "metadataoptions": map[string]interface{}{
                        "sentenceBoundaryEnabled": false,
                        "wordBoundaryEnabled":     true,
                    },
                    "outputFormat": "audio-24khz-48kbitrate-mono-mp3",
                },
            },
        },
    }
    
    configJSON, _ := json.Marshal(config)
    return fmt.Sprintf("X-Timestamp:%sZ\r\nContent-Type:application/json; charset=utf-8\r\nPath:speech.config\r\n\r\n%s", time.Now().UTC().Format(time.RFC3339), string(configJSON))
}

func BuildTTSSpeechMessage(reqID, ssml string) string {
	return fmt.Sprintf("X-RequestId:%s\r\nContent-Type:application/ssml+xml\r\nX-Timestamp:%sZ\r\nPath:ssml\r\n\r\n%s", reqID, time.Now().UTC().Format(time.RFC3339), ssml)
}

func (vr *VoiceRegistry) SetVoiceRegistry(voice string) {
	vr.Mu.Lock()
	defer vr.Mu.Unlock()
	vr.Voices[voice] = true
}

func (vr *VoiceRegistry) CheckVoiceRegistry(voice string) bool {
	vr.Mu.Lock()
	defer vr.Mu.Unlock()
	return vr.Voices[voice]
}

func InitVoices() error {
	voices, err := GetVoices()
	if err != nil {
		return errors.Join(ErrServiceFailure, fmt.Errorf("failed to initialize voices: %w", err))
	}

	for _, voice := range voices {
		ValidVoices.SetVoiceRegistry(voice.ShortName)
	}

	return nil
}

func ValidateNumbers(value, suffix, errMsg string) error {
	if !regexp.MustCompile(fmt.Sprintf(`^-?\d{1,3}%s$`, suffix)).MatchString(value) {
		return errors.Join(ErrInvalidInput, fmt.Errorf("invalid format: %s", errMsg))
	}

	numStr := strings.TrimSuffix(value, suffix)
	num, _ := strconv.Atoi(numStr)
	
	if num < minValue || num > maxValue {
		return errors.Join(ErrInvalidInput, fmt.Errorf("value out of range: %s", errMsg))
	}

	return nil
}

func ValidateVoice(voice string) error {
	if !ValidVoices.CheckVoiceRegistry(voice) {
		return errors.Join(ErrInvalidInput, fmt.Errorf("invalid voice: %s", voice))
	}

	return nil
}

func ValidateText(text string) error {
	if strings.TrimSpace(text) == "" {
		return errors.Join(ErrInvalidInput, errors.New("text cannot be empty"))
	}

	return nil
}

func ValidatePitch(pitch string) error {
	return ValidateNumbers(pitch, "Hz", "pitch must be between -100Hz and 100Hz")
}

func ValidateRate(rate string) error {
	return ValidateNumbers(rate, "%", "rate must be between -100% and 100%")
}

func ValidateVolume(volume string) error {
	return ValidateNumbers(volume, "%", "volume must be between -100% and 100%")
}

func (r *Request) Validate() error {
	var errs []error

	if err := ValidateVoice(r.Voice); err != nil {
		errs = append(errs, err)
	}
	if err := ValidateText(r.Text); err != nil {
		errs = append(errs, err)
	}
	if err := ValidatePitch(r.Pitch); err != nil {
		errs = append(errs, err)
	}
	if err := ValidateRate(r.Rate); err != nil {
		errs = append(errs, err)
	}
	if err := ValidateVolume(r.Volume); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (r *Request) SetRequestDefaults() {
	if r.Voice == "" {
		r.Voice = "en-US-AriaNeural"
	}
	if r.Pitch == "" {
		r.Pitch = "0Hz"
	}
	if r.Rate == "" {
		r.Rate = "0%"
	}
	if r.Volume == "" {
		r.Volume = "0%"
	}

	if !strings.HasSuffix(r.Rate, "%") {
		r.Rate = r.Rate + "%"
	}
	if !strings.HasSuffix(r.Volume, "%") {
		r.Volume = r.Volume + "%"
	}
	if !strings.HasSuffix(r.Pitch, "Hz") {
		r.Pitch += "Hz"
	}
}