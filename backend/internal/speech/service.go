package speech

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
    "errors"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func GetVoices() ([]Voice, error) {
	url := fmt.Sprintf("%s?trustedclienttoken=%s", VoicesURL, TrustedClientToken)

	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.Join(ErrNetworkFailure, fmt.Errorf("failed to fetch voices: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Join(ErrServiceFailure, fmt.Errorf("service returned status %d", resp.StatusCode))
	}

	var voices []Voice
	if err := json.NewDecoder(resp.Body).Decode(&voices); err != nil {
		return nil, errors.Join(ErrServiceFailure, fmt.Errorf("failed to decode voices response: %w", err))
	}

	return voices, nil
}

func (r *Request) GetSSML() (string, error) {
	r.SetRequestDefaults()

	if err := r.Validate(); err != nil {
		return "", err
	}

	ssml := fmt.Sprintf(`<speak version='1.0' xml:lang='en-US'><voice name='%s'><prosody pitch='%s' rate='%s' volume='%s'>%s</prosody></voice></speak>`,
		r.Voice, r.Pitch, r.Rate, r.Volume, r.Text)

	return ssml, nil
}

func ReadAudioResponse(c *websocket.Conn) ([]byte, error) {
	var audioBuffer bytes.Buffer

	for {
		messageType, message, err := c.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				break
			}
			return nil, errors.Join(ErrSynthesis, fmt.Errorf("failed to read message: %w", err))
		}

		switch {
		case messageType == websocket.BinaryMessage:
			if bytes.Contains(message, []byte("Path:audio\r\n")) {
				audioData := bytes.SplitN(message, []byte("Path:audio\r\n"), 2)[1]
				if _, err := audioBuffer.Write(audioData); err != nil {
					return nil, errors.Join(ErrSynthesis, fmt.Errorf("failed to write audio data: %w", err))
				}
			}
		case bytes.Contains(message, []byte("Path:turn.end")):
			return audioBuffer.Bytes(), nil
		case bytes.Contains(message, []byte("Path:error")):
			return nil, errors.Join(ErrSynthesis, fmt.Errorf("service error: %s", string(message)))
		}
	}

	if audioBuffer.Len() == 0 {
		return nil, ErrNoAudio
	}

	return audioBuffer.Bytes(), nil
}

func (r *Request) Synthesize() ([]byte, error) {
	ssml, err := r.GetSSML()
	if err != nil {
		return nil, err
	}

	reqID := uuid.New().String()
	u, err := url.Parse(WssURL)
	if err != nil {
		return nil, errors.Join(ErrServiceFailure, fmt.Errorf("invalid service URL: %w", err))
	}

	q := u.Query()
	q.Set("trustedclienttoken", TrustedClientToken)
	q.Set("ConnectionId", reqID)
	u.RawQuery = q.Encode()

	c, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		if resp != nil {
			return nil, errors.Join(ErrNetworkFailure, fmt.Errorf("websocket connection failed with status %d: %w", resp.StatusCode, err))
		}
		return nil, errors.Join(ErrNetworkFailure, fmt.Errorf("websocket connection failed: %w", err))
	}
	defer c.Close()

	if err := c.WriteMessage(websocket.TextMessage, []byte(BuildTTSConfigMessage())); err != nil {
		return nil, errors.Join(ErrSynthesis, fmt.Errorf("failed to send config: %w", err))
	}

	speechMessage := BuildTTSSpeechMessage(reqID, ssml)
	if err := c.WriteMessage(websocket.TextMessage, []byte(speechMessage)); err != nil {
		return nil, errors.Join(ErrSynthesis, fmt.Errorf("failed to send SSML: %w", err))
	}

	audio, err := ReadAudioResponse(c)
	if err != nil {
		return nil, err
	}

	return audio, nil
}