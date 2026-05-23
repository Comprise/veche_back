package livekit

import (
	"time"

	"github.com/livekit/protocol/auth"
)

type TokenService struct {
	apiKey    string
	apiSecret string
}

func NewTokenService(key, secret string) *TokenService {
	return &TokenService{key, secret}
}

func (s *TokenService) BuildConnectionDetails(
	roomName,
	participantName,
	metadata,
	identity string,
) (
	token string,
	err error,
) {
	at := auth.NewAccessToken(s.apiKey, s.apiSecret)

	t := true
	grant := &auth.VideoGrant{
		RoomJoin:       true,
		Room:           roomName,
		CanPublish:     &t,
		CanPublishData: &t,
		CanSubscribe:   &t,
	}

	at.SetVideoGrant(grant).
		SetIdentity(identity).
		SetName(participantName).
		SetMetadata(metadata).
		SetValidFor(5 * time.Minute)

	return at.ToJWT()
}
