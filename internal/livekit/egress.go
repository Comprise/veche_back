package livekit

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

// Recorder is the interface room handlers depend on.
type Recorder interface {
	StartRecording(roomName string) error
	StopRecording(roomName string) error
}

type EgressService struct {
	client *lksdk.EgressClient
}

func NewEgressService(apiKey, apiSecret, livekitURL string) *EgressService {
	host := hostFromURL(livekitURL)
	return &EgressService{
		client: lksdk.NewEgressClient(fmt.Sprintf("%s:443", host), apiKey, apiSecret),
	}
}

func hostFromURL(rawURL string) string {
	if !strings.HasPrefix(rawURL, "http") && !strings.HasPrefix(rawURL, "wss") {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Hostname()
}

func (s *EgressService) StartRecording(roomName string) error {
	active, err := s.getActiveEgresses(roomName)
	if err != nil {
		return fmt.Errorf("failed to list egress: %w", err)
	}
	if len(active) > 0 {
		return ErrRecordingAlreadyActive
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fileName := fmt.Sprintf("%s-%s.mp3", time.Now().Format(time.RFC3339), roomName)
	req := &livekit.RoomCompositeEgressRequest{
		RoomName:  roomName,
		AudioOnly: true,
		Output: &livekit.RoomCompositeEgressRequest_File{
			File: &livekit.EncodedFileOutput{
				Filepath: fileName,
				FileType: livekit.EncodedFileType_MP3,
			},
		},
		Options: &livekit.RoomCompositeEgressRequest_Advanced{
			Advanced: &livekit.EncodingOptions{
				AudioBitrate:   64,
				AudioFrequency: 22050,
				AudioCodec:     livekit.AudioCodec_AC_MP3,
			},
		},
	}

	_, err = s.client.StartRoomCompositeEgress(ctx, req)
	if err != nil {
		return fmt.Errorf("start egress failed: %w", err)
	}
	return nil
}

func (s *EgressService) StopRecording(roomName string) error {
	active, err := s.getActiveEgresses(roomName)
	if err != nil {
		return fmt.Errorf("list egress failed: %w", err)
	}
	if len(active) == 0 {
		return ErrNoActiveRecording
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, eg := range active {
		if _, err = s.client.StopEgress(ctx, &livekit.StopEgressRequest{EgressId: eg.EgressId}); err != nil {
			return fmt.Errorf("failed to stop egress %s: %w", eg.EgressId, err)
		}
	}
	return nil
}

func (s *EgressService) getActiveEgresses(roomName string) ([]*livekit.EgressInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := s.client.ListEgress(ctx, &livekit.ListEgressRequest{
		RoomName: roomName,
		Active:   true,
	})
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}
