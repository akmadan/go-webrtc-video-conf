package websocket

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

var roomIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,64}$`)

func validateMessage(msg Message) error {
	if msg.Type == "" {
		return errors.New("type is required")
	}

	switch msg.Type {
	case MessageTypeJoin:
		if !roomIDPattern.MatchString(msg.RoomID) {
			return errors.New("roomId must be 3-64 chars and only contain letters, numbers, - or _")
		}
		if len(msg.Data) > 0 {
			var join JoinData
			if err := json.Unmarshal(msg.Data, &join); err != nil {
				return errors.New("invalid join data")
			}
			if len(strings.TrimSpace(join.Name)) > 64 {
				return errors.New("name must be at most 64 characters")
			}
		}
	case MessageTypeLeave:
		return nil
	case MessageTypeOffer, MessageTypeAnswer:
		if len(msg.Data) == 0 {
			return errors.New("data is required")
		}
		var sdp SDPData
		if err := json.Unmarshal(msg.Data, &sdp); err != nil {
			return errors.New("invalid SDP data")
		}
		if strings.TrimSpace(sdp.SDP) == "" {
			return errors.New("sdp is required")
		}
		if sdp.Type != string(msg.Type) {
			return errors.New("sdp type must match message type")
		}
	case MessageTypeICECandidate:
		if len(msg.Data) == 0 {
			return errors.New("data is required")
		}
		var candidate ICECandidateData
		if err := json.Unmarshal(msg.Data, &candidate); err != nil {
			return errors.New("invalid ICE candidate data")
		}
		if strings.TrimSpace(candidate.Candidate) == "" {
			return errors.New("candidate is required")
		}
	default:
		return errors.New("unknown message type")
	}

	return nil
}

