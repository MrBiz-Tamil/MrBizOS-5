package livekit

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/livekit/protocol/auth"
	// lksdk "github.com/livekit/server-sdk-go/v2"
)

type LiveKitService struct {
	// roomClient   *lksdk.RoomServiceClient
	// egressClient *lksdk.EgressServiceClient
	apiKey    string
	apiSecret string
}

func boolPtr(b bool) *bool {
	return &b
}

var instance *LiveKitService

func GetService() *LiveKitService {
	if instance == nil {
		InitService()
	}
	return instance
}

// GetServerURL returns the LiveKit WebSocket URL (wss://) for the frontend.
// Prefers LIVEKIT_WS_URL; falls back to LIVEKIT_HTTP_URL with scheme conversion.
func GetServerURL() string {
	if u := os.Getenv("LIVEKIT_WS_URL"); u != "" {
		return u
	}
	u := os.Getenv("LIVEKIT_HTTP_URL")
	// LiveKit clients require wss:// (or ws://); convert https:// / http:// automatically.
	u = strings.Replace(u, "https://", "wss://", 1)
	u = strings.Replace(u, "http://", "ws://", 1)
	return u
}

func InitService() {
	apiKey := os.Getenv("LIVEKIT_API_KEY")
	apiSecret := os.Getenv("LIVEKIT_API_SECRET")
	httpUrl := os.Getenv("LIVEKIT_HTTP_URL")

	if apiKey == "" || apiSecret == "" || httpUrl == "" {
		log.Println("WARNING: LIVEKIT_API_KEY, LIVEKIT_API_SECRET, or LIVEKIT_HTTP_URL not set. LiveKit features will be disabled.")
		return
	}

	// roomClient := lksdk.NewRoomServiceClient(httpUrl, apiKey, apiSecret)
	// egressClient := lksdk.NewEgressServiceClient(httpUrl, apiKey, apiSecret)

	instance = &LiveKitService{
		// roomClient:   roomClient,
		// egressClient: egressClient,
		apiKey:    apiKey,
		apiSecret: apiSecret,
	}
}

func (s *LiveKitService) GenerateToken(roomName string, identity string, name string, role string) (string, error) {
	if s == nil || s.apiKey == "" {
		return "", errors.New("livekit not initialized")
	}

	at := auth.NewAccessToken(s.apiKey, s.apiSecret)
	grant := &auth.VideoGrant{
		RoomJoin:     true,
		Room:         roomName,
		CanPublish:   boolPtr(true),
		CanSubscribe: boolPtr(true),
	}

	if role == "doctor" || role == "chief_doctor" || role == "hospital_head_doctor" {
		grant.RoomCreate = true
		grant.RoomAdmin = true
		grant.RoomRecord = true
	} else if role == "admin" || role == "super_admin" {
		grant.RoomAdmin = true
		grant.RoomRecord = true
		grant.CanPublish = boolPtr(false)
	} else {
		// Patient
		grant.RoomCreate = false
		grant.RoomAdmin = false
		grant.RoomRecord = false
	}

	at.AddGrant(grant).
		SetIdentity(identity).
		SetName(name).
		SetValidFor(time.Hour)

	return at.ToJWT()
}

func (s *LiveKitService) StartRecording(roomName string) (string, error) {
	// Implement based on Egress Client (requires proto definitions usually, skipping complex egress for now to keep migration simple)
	// Returning mock ID
	if s == nil {
		return "", errors.New("livekit not initialized")
	}
	return "mock_recording_" + fmt.Sprintf("%d", time.Now().Unix()), nil
}

func (s *LiveKitService) StopRecording(egressID string) error {
	if s == nil {
		return errors.New("livekit not initialized")
	}
	return nil
}
