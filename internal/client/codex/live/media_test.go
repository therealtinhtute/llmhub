package live

import (
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"
	"github.com/therealtinhtute/llmhub/internal/runtimecontrol"
)

func TestPionMediaRelaySelectsRemoteProxyMode(t *testing.T) {
	clientAPI := newTestWebRTCAPI(t)
	client, errClient := clientAPI.NewPeerConnection(webrtc.Configuration{})
	if errClient != nil {
		t.Fatalf("create client PeerConnection: %v", errClient)
	}
	defer closeTestPeerConnection(t, client)
	if _, errChannel := client.CreateDataChannel(realtimeDataChannelLabel, nil); errChannel != nil {
		t.Fatalf("create client DataChannel: %v", errChannel)
	}
	clientOffer := completeOffer(t, client)
	relay, errRelay := NewPionMediaRelay(runtimecontrol.CodexLiveSettings{Enabled: true})
	if errRelay != nil {
		t.Fatalf("create media relay: %v", errRelay)
	}

	for name, testCase := range map[string]struct {
		proxyURL string
		proxied  bool
	}{
		"inherit": {proxyURL: ""},
		"direct":  {proxyURL: "direct"},
		"HTTP":    {proxyURL: "http://proxy.example:8080", proxied: true},
		"HTTPS":   {proxyURL: "https://proxy.example:8443", proxied: true},
		"SOCKS5":  {proxyURL: "socks5://proxy.example:1080", proxied: true},
		"SOCKS5H": {proxyURL: "socks5h://proxy.example:1080", proxied: true},
	} {
		t.Run(name, func(t *testing.T) {
			session, upstreamOffer, errSession := relay.NewSession(t.Context(), clientOffer, MediaSessionRoute{ProxyURL: testCase.proxyURL})
			if errSession != nil {
				t.Fatalf("create media session: %v", errSession)
			}
			pionSession, ok := session.(*pionMediaSession)
			if !ok {
				t.Fatalf("media session type = %T", session)
			}
			if got := pionSession.proxyDialer != nil; got != testCase.proxied {
				t.Fatalf("proxied = %t, want %t", got, testCase.proxied)
			}
			if testCase.proxied && !offerCandidatesAreLoopback(t, upstreamOffer) {
				t.Fatal("proxied upstream offer exposed a non-loopback candidate")
			}
			if errClose := session.Close(); errClose != nil {
				t.Fatalf("close media session: %v", errClose)
			}
		})
	}

	if _, _, errSession := relay.NewSession(t.Context(), clientOffer, MediaSessionRoute{ProxyURL: "invalid-proxy"}); errSession == nil {
		t.Fatal("expected invalid proxy URL to fail media session creation")
	}
}

func TestIsPublicRemoteIP(t *testing.T) {
	for rawIP, want := range map[string]bool{
		"8.8.8.8":      true,
		"2001:4860::1": true,
		"127.0.0.1":    false,
		"10.0.0.1":     false,
		"169.254.1.1":  false,
		"224.0.0.1":    false,
		"::1":          false,
		"fc00::1":      false,
		"fe80::1":      false,
		"ff02::1":      false,
		"0.0.0.0":      false,
	} {
		if got := isPublicRemoteIP(net.ParseIP(rawIP)); got != want {
			t.Errorf("isPublicRemoteIP(%q) = %t, want %t", rawIP, got, want)
		}
	}
	if isPublicRemoteIP(nil) {
		t.Fatal("isPublicRemoteIP(nil) = true, want false")
	}
}

func offerCandidatesAreLoopback(t *testing.T, offer string) bool {
	t.Helper()
	lines := strings.Split(strings.ReplaceAll(offer, "\r\n", "\n"), "\n")
	candidateCount := 0
	for _, line := range lines {
		if !strings.HasPrefix(line, "a=candidate:") {
			continue
		}
		candidateCount++
		fields := strings.Fields(strings.TrimPrefix(line, "a=candidate:"))
		if len(fields) < 6 {
			t.Fatalf("malformed offer candidate: %q", line)
		}
		address := net.ParseIP(fields[4])
		if address == nil || !address.IsLoopback() {
			return false
		}
	}
	return candidateCount > 0
}

func newTestWebRTCAPI(t *testing.T) *webrtc.API {
	t.Helper()
	mediaEngine := &webrtc.MediaEngine{}
	if errRegister := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{RTPCodecCapability: opusCodec, PayloadType: 111}, webrtc.RTPCodecTypeAudio); errRegister != nil {
		t.Fatalf("register test Opus codec: %v", errRegister)
	}
	interceptorRegistry := &interceptor.Registry{}
	if errRegister := webrtc.RegisterDefaultInterceptors(mediaEngine, interceptorRegistry); errRegister != nil {
		t.Fatalf("register test interceptors: %v", errRegister)
	}
	return webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine), webrtc.WithInterceptorRegistry(interceptorRegistry))
}

func completeOffer(t *testing.T, connection *webrtc.PeerConnection) string {
	t.Helper()
	gatherComplete := webrtc.GatheringCompletePromise(connection)
	offer, errOffer := connection.CreateOffer(nil)
	if errOffer != nil {
		t.Fatalf("create offer: %v", errOffer)
	}
	if errLocal := connection.SetLocalDescription(offer); errLocal != nil {
		t.Fatalf("set local offer: %v", errLocal)
	}
	select {
	case <-gatherComplete:
	case <-t.Context().Done():
		t.Fatal("offer ICE gathering did not complete")
	}
	return connection.LocalDescription().SDP
}

func closeTestPeerConnection(t *testing.T, connection *webrtc.PeerConnection) {
	t.Helper()
	if errClose := connection.Close(); errClose != nil {
		t.Errorf("close test PeerConnection: %v", errClose)
	}
}

func TestMediaForwardingLogRedactsCredentialSurface(t *testing.T) {
	session := &pionMediaSession{mediaSessionID: "media-session", proxyScheme: "http", credential: "Voice credential", authIndex: "auth-index"}
	session.proxyDialer = &recordingProxyDialer{dials: make(chan recordedProxyDial, 1)}
	fields := fmt.Sprint(session.forwardingLogFields())
	for _, secret := range []string{"user:secret", "proxy.example"} {
		if strings.Contains(fields, secret) {
			t.Fatalf("forwarding log leaked %q: %s", secret, fields)
		}
	}
}
