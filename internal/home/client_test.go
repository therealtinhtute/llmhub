package home

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/therealtinhtute/llmhub/internal/config"
)

func TestAuthDispatchRequestIncludesCount(t *testing.T) {
	req := newAuthDispatchRequest("gpt-5.4", "session-1", http.Header{"Authorization": {"Bearer test"}}, 2)

	raw, err := json.Marshal(&req)
	if err != nil {
		t.Fatalf("marshal auth dispatch request: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal auth dispatch request: %v", err)
	}
	if got := int(payload["count"].(float64)); got != 2 {
		t.Fatalf("count = %d, want 2", got)
	}
}

func TestAuthDispatchRequestDefaultsCountToOne(t *testing.T) {
	req := newAuthDispatchRequest("gpt-5.4", "", nil, 0)

	if req.Count != 1 {
		t.Fatalf("count = %d, want 1", req.Count)
	}
}

func TestRedisOptionsHomeTLSDisabled(t *testing.T) {
	client := New(config.HomeConfig{
		Enabled: true,
		Host:    "127.0.0.1",
		Port:    6379,
	})

	client.mu.Lock()
	options, err := client.redisOptionsLocked("127.0.0.1:6379")
	client.mu.Unlock()
	if err != nil {
		t.Fatalf("redisOptionsLocked() error = %v", err)
	}

	if options.TLSConfig != nil {
		t.Fatalf("TLSConfig = %#v, want nil", options.TLSConfig)
	}
	if options.Password != "" {
		t.Fatalf("Password = %q, want empty", options.Password)
	}
}

func TestRedisOptionsHomeTLSEnabledUsesSeedHostAsServerName(t *testing.T) {
	client := New(config.HomeConfig{
		Enabled: true,
		Host:    "home.example.com",
		Port:    444,
		TLS: config.HomeTLSConfig{
			Enable: true,
		},
	})
	client.homeCfg.Host = "127.0.0.1"

	client.mu.Lock()
	options, err := client.redisOptionsLocked("127.0.0.1:444")
	client.mu.Unlock()
	if err != nil {
		t.Fatalf("redisOptionsLocked() error = %v", err)
	}

	if options.TLSConfig == nil {
		t.Fatal("TLSConfig is nil")
	}
	if options.TLSConfig.ServerName != "home.example.com" {
		t.Fatalf("ServerName = %q, want home.example.com", options.TLSConfig.ServerName)
	}
	if options.TLSConfig.MinVersion != tls.VersionTLS12 {
		t.Fatalf("MinVersion = %d, want TLS 1.2", options.TLSConfig.MinVersion)
	}
}

func TestRedisOptionsHomeTLSEnabledUsesExplicitServerName(t *testing.T) {
	client := New(config.HomeConfig{
		Enabled: true,
		Host:    "127.0.0.1",
		Port:    444,
		TLS: config.HomeTLSConfig{
			Enable:             true,
			ServerName:         "home.example.com",
			InsecureSkipVerify: true,
		},
	})

	client.mu.Lock()
	options, err := client.redisOptionsLocked("127.0.0.1:444")
	client.mu.Unlock()
	if err != nil {
		t.Fatalf("redisOptionsLocked() error = %v", err)
	}

	if options.TLSConfig == nil {
		t.Fatal("TLSConfig is nil")
	}
	if options.TLSConfig.ServerName != "home.example.com" {
		t.Fatalf("ServerName = %q, want home.example.com", options.TLSConfig.ServerName)
	}
	if !options.TLSConfig.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify = false, want true")
	}
}

func TestRefreshClusterNodesDisabledSkipsRedisCommand(t *testing.T) {
	client := New(config.HomeConfig{
		Enabled:                 true,
		Host:                    "127.0.0.1",
		Port:                    1,
		DisableClusterDiscovery: true,
	})

	switched, err := client.refreshClusterNodes(context.Background())
	if err != nil {
		t.Fatalf("refreshClusterNodes() error = %v", err)
	}
	if switched {
		t.Fatal("refreshClusterNodes() switched = true, want false")
	}
	if client.cmd != nil || client.sub != nil {
		t.Fatalf("redis clients were initialized when cluster discovery was disabled")
	}
}

func TestFailoverAfterReconnectFailureDisabledDoesNotSwitchToClusterNode(t *testing.T) {
	client := New(config.HomeConfig{
		Enabled:                 true,
		Host:                    "seed.example.com",
		Port:                    8327,
		DisableClusterDiscovery: true,
	})
	client.mu.Lock()
	client.clusterNodes = []clusterNode{{IP: "other.example.com", Port: 8327}}
	client.reconnectFailures = homeReconnectFailoverThreshold - 1
	client.mu.Unlock()

	switched, addr := client.failoverAfterReconnectFailure()
	if switched {
		t.Fatalf("failoverAfterReconnectFailure() switched to %s, want no switch", addr)
	}
	if got, _ := client.addr(); got != "seed.example.com:8327" {
		t.Fatalf("addr() = %q, want seed.example.com:8327", got)
	}
}

func TestSubscriptionParametersIncludeMembershipState(t *testing.T) {
	client := New(config.HomeConfig{Enabled: true, Host: "home.example.com", Port: 8327})
	if errSet := client.SetLifecycleConfig(config.CredentialConcurrencyConfig{
		LifecycleConfigRevision: 7,
		CPAHeartbeatTimeout:     1500 * time.Millisecond,
		CPACancelBound:          time.Second,
		ReclaimGrace:            time.Second,
		CleanupInterval:         time.Second,
		ReleaseFlushInterval:    time.Millisecond,
		ReleaseMaxBackoff:       time.Millisecond,
		BusyRetryMin:            time.Millisecond,
		BusyRetryMax:            time.Millisecond,
		MaxLimit:                1,
	}); errSet != nil {
		t.Fatal(errSet)
	}

	args, timeout := client.subscriptionParameters()
	if want := []string{redisChannelConfig, "7", client.MembershipInstanceID()}; !reflect.DeepEqual(args, want) {
		t.Fatalf("subscription args = %#v, want %#v", args, want)
	}
	if timeout != 1500*time.Millisecond {
		t.Fatalf("subscription timeout = %v, want 1.5s", timeout)
	}

	client.markMembershipTakeoverEligible()
	args, _ = client.subscriptionParameters()
	if want := []string{redisChannelConfig, "7", "takeover", client.MembershipInstanceID()}; !reflect.DeepEqual(args, want) {
		t.Fatalf("takeover subscription args = %#v, want %#v", args, want)
	}

	client.EnableLegacyMembership()
	args, _ = client.subscriptionParameters()
	if want := []string{redisChannelConfig, "7"}; !reflect.DeepEqual(args, want) {
		t.Fatalf("legacy subscription args = %#v, want %#v", args, want)
	}
}

func TestNewLifetimePreservesMembershipAndClusterState(t *testing.T) {
	client := New(config.HomeConfig{Enabled: true, Host: "seed.example.com", Port: 8327})
	client.mu.Lock()
	client.clusterNodes = []clusterNode{{IP: "other.example.com", Port: 8327}}
	client.reconnectFailures = 2
	client.legacyMembership = true
	client.mu.Unlock()
	client.recoveryState.Store(uint32(recoveryStateSwitchingTakeover))

	next := client.NewLifetime()
	if next == nil {
		t.Fatal("NewLifetime() = nil")
	}
	if next.MembershipInstanceID() != client.MembershipInstanceID() {
		t.Fatalf("instance id = %q, want %q", next.MembershipInstanceID(), client.MembershipInstanceID())
	}
	if !next.LegacyMembership() {
		t.Fatal("legacy membership was not preserved")
	}
	if got := recoveryState(next.recoveryState.Load()); got != recoveryStateSwitchingTakeover {
		t.Fatalf("recovery state = %d, want %d", got, recoveryStateSwitchingTakeover)
	}
	if switched, addr := next.failoverAfterReconnectFailure(); !switched || addr != "other.example.com:8327" {
		t.Fatalf("failover = %v addr %q, want switch to preserved cluster node", switched, addr)
	}
}

func TestConcurrencyReleaseDoesNotOpenBeforeMembershipReady(t *testing.T) {
	tests := []struct {
		name  string
		state recoveryState
	}{
		{name: "takeover pending", state: recoveryStateTakeoverEligible},
		{name: "target switching", state: recoveryStateSwitching},
		{name: "target switching with takeover", state: recoveryStateSwitchingTakeover},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			client := New(config.HomeConfig{Enabled: true, Host: "next.example.com", Port: 8327})
			client.recoveryState.Store(uint32(testCase.state))
			errRelease := client.PushConcurrencyRelease(context.Background(), ConcurrencyReleaseFrame{CredentialID: "cred-a", Model: "model-a", ReleaseSeq: 1})
			if !errors.Is(errRelease, ErrNotConnected) {
				t.Fatalf("PushConcurrencyRelease() error = %v, want %v", errRelease, ErrNotConnected)
			}
			client.mu.Lock()
			releaseClient := client.release
			client.mu.Unlock()
			if releaseClient != nil {
				t.Fatal("release client was opened before the membership became ready")
			}
		})
	}
}

func TestAmbiguousDispatchSuppressesTakeoverForNextLifetime(t *testing.T) {
	client := New(config.HomeConfig{Enabled: true, Host: "next.example.com", Port: 8327})
	client.recoveryState.Store(uint32(recoveryStateSwitchingTakeover))
	client.AbortAmbiguousDispatch()
	if !client.AmbiguousDispatch() {
		t.Fatal("AmbiguousDispatch() = false, want true")
	}
	next := client.NewLifetime()
	if got := recoveryState(next.recoveryState.Load()); got != recoveryStateSwitching {
		t.Fatalf("next recovery state = %d, want %d", got, recoveryStateSwitching)
	}
}

func TestAbortAmbiguousDispatchFencesClientReuse(t *testing.T) {
	client := New(config.HomeConfig{Enabled: true, Host: "next.example.com", Port: 8327})
	client.AbortAmbiguousDispatch()

	if _, errClient := client.commandClient(); !errors.Is(errClient, ErrDispatchFenced) {
		t.Fatalf("commandClient() error = %v, want %v", errClient, ErrDispatchFenced)
	}
	if _, errClient := client.subscriptionClient(); !errors.Is(errClient, ErrDispatchFenced) {
		t.Fatalf("subscriptionClient() error = %v, want %v", errClient, ErrDispatchFenced)
	}
	errRelease := client.PushConcurrencyRelease(context.Background(), ConcurrencyReleaseFrame{CredentialID: "cred-a", Model: "model-a", ReleaseSeq: 1})
	if !errors.Is(errRelease, ErrDispatchFenced) {
		t.Fatalf("PushConcurrencyRelease() error = %v, want %v", errRelease, ErrDispatchFenced)
	}
	client.mu.Lock()
	cmd := client.cmd
	sub := client.sub
	release := client.release
	client.mu.Unlock()
	if cmd != nil || sub != nil || release != nil {
		t.Fatal("fenced client opened Redis clients")
	}
}

func TestKVCompareAndSwapSendsCASCommand(t *testing.T) {
	client, commands := newRedisCommandTestClient(t, func(args []string) string {
		if len(args) > 0 && strings.EqualFold(args[0], "CAS") {
			return ":1\r\n"
		}
		return "-ERR unexpected command\r\n"
	})

	swapped, errCAS := client.KVCompareAndSwap(context.Background(), "key", []byte("old"), true, []byte("new"), 1500*time.Millisecond)
	if errCAS != nil {
		t.Fatalf("KVCompareAndSwap() error = %v", errCAS)
	}
	if !swapped {
		t.Fatal("KVCompareAndSwap() swapped = false, want true")
	}
	want := []string{"CAS", "key", "1", "old", "new", "PX", "1500"}
	if lastCommand := commands.Last(); !reflect.DeepEqual(lastCommand, want) {
		t.Fatalf("last command = %#v, want %#v", lastCommand, want)
	}
}

func TestKVCompareAndSwapOmitsPXWithoutTTL(t *testing.T) {
	client, commands := newRedisCommandTestClient(t, func(args []string) string {
		if len(args) > 0 && strings.EqualFold(args[0], "CAS") {
			return ":1\r\n"
		}
		return "-ERR unexpected command\r\n"
	})

	if _, errCAS := client.KVCompareAndSwap(context.Background(), "key", nil, false, []byte("new"), 0); errCAS != nil {
		t.Fatalf("KVCompareAndSwap() error = %v", errCAS)
	}
	want := []string{"CAS", "key", "0", "", "new"}
	if lastCommand := commands.Last(); !reflect.DeepEqual(lastCommand, want) {
		t.Fatalf("last command = %#v, want %#v", lastCommand, want)
	}
}

func TestKVCompareAndSwapReportsMismatch(t *testing.T) {
	client, _ := newRedisCommandTestClient(t, func(args []string) string {
		if len(args) > 0 && strings.EqualFold(args[0], "CAS") {
			return ":0\r\n"
		}
		return "-ERR unexpected command\r\n"
	})

	swapped, errCAS := client.KVCompareAndSwap(context.Background(), "key", []byte("old"), true, []byte("new"), time.Minute)
	if errCAS != nil {
		t.Fatalf("KVCompareAndSwap() error = %v", errCAS)
	}
	if swapped {
		t.Fatal("KVCompareAndSwap() swapped = true, want false")
	}
}

func TestKVCompareAndSwapLatchesUnsupportedHome(t *testing.T) {
	client, commands := newRedisCommandTestClient(t, func(args []string) string {
		if len(args) > 0 && strings.EqualFold(args[0], "CAS") {
			return "-ERR unknown command 'cas'\r\n"
		}
		return "-ERR unexpected command\r\n"
	})

	_, errFirst := client.KVCompareAndSwap(context.Background(), "key", nil, false, []byte("new"), time.Minute)
	if !errors.Is(errFirst, ErrCompareAndSwapUnsupported) {
		t.Fatalf("KVCompareAndSwap() first error = %v, want ErrCompareAndSwapUnsupported", errFirst)
	}
	if sent := commands.CountCommandKey("CAS", "key"); sent != 1 {
		t.Fatalf("CAS sent %d times, want 1", sent)
	}

	_, errSecond := client.KVCompareAndSwap(context.Background(), "key", nil, false, []byte("new"), time.Minute)
	if !errors.Is(errSecond, ErrCompareAndSwapUnsupported) {
		t.Fatalf("KVCompareAndSwap() second error = %v, want ErrCompareAndSwapUnsupported", errSecond)
	}
	if sent := commands.CountCommandKey("CAS", "key"); sent != 1 {
		t.Fatalf("CAS sent %d times after latching, want 1", sent)
	}
}

func TestMembershipTakeoverUnavailableError(t *testing.T) {
	if !IsMembershipTakeoverUnavailableError(errors.New("ERR membership_takeover_unavailable")) {
		t.Fatal("takeover unavailable error was not recognized")
	}
	if IsMembershipTakeoverUnavailableError(errors.New("ERR wrong number of arguments for 'subscribe' command")) {
		t.Fatal("legacy protocol error was recognized as takeover unavailable")
	}
	if !IsLegacyMembershipProtocolError(errors.New("ERR wrong number of arguments for 'subscribe' command")) {
		t.Fatal("legacy protocol error was not recognized")
	}
	for _, errUnrelated := range []error{errors.New("ERR connection refused"), errors.New("ERR duplicate certificate"), context.DeadlineExceeded} {
		if IsMembershipTakeoverUnavailableError(errUnrelated) || IsLegacyMembershipProtocolError(errUnrelated) {
			t.Fatalf("unrelated error %q was classified as a membership protocol error", errUnrelated)
		}
	}
}

type redisCommandLog struct {
	mu       sync.Mutex
	commands [][]string
}

func (l *redisCommandLog) Append(args []string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.commands = append(l.commands, append([]string(nil), args...))
}

func (l *redisCommandLog) Last() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.commands) == 0 {
		return nil
	}
	return append([]string(nil), l.commands[len(l.commands)-1]...)
}

func (l *redisCommandLog) CountCommandKey(commandName string, key string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	count := 0
	for _, command := range l.commands {
		if len(command) >= 2 && strings.EqualFold(command[0], commandName) && command[1] == key {
			count++
		}
	}
	return count
}

const homeRedisTestOperationTimeout = 50 * time.Millisecond

func newRedisCommandTestClient(t *testing.T, handler func([]string) string) (*Client, *redisCommandLog) {
	t.Helper()

	listener, errListen := net.Listen("tcp", "127.0.0.1:0")
	if errListen != nil {
		t.Fatalf("listen: %v", errListen)
	}
	log := &redisCommandLog{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, errAccept := listener.Accept()
			if errAccept != nil {
				return
			}
			go serveRedisCommandTestConn(conn, log, handler)
		}
	}()
	t.Cleanup(func() {
		_ = listener.Close()
		<-done
	})

	host, portText, errSplit := net.SplitHostPort(listener.Addr().String())
	if errSplit != nil {
		t.Fatalf("split listener addr: %v", errSplit)
	}
	port, errPort := strconv.Atoi(portText)
	if errPort != nil {
		t.Fatalf("parse listener port: %v", errPort)
	}
	client := New(config.HomeConfig{
		Enabled:                 true,
		Host:                    host,
		Port:                    port,
		DisableClusterDiscovery: true,
	})
	client.cmd = redis.NewClient(&redis.Options{
		Addr:                  listener.Addr().String(),
		Protocol:              2,
		DisableIdentity:       true,
		DialTimeout:           homeRedisTestOperationTimeout,
		ReadTimeout:           homeRedisTestOperationTimeout,
		WriteTimeout:          homeRedisTestOperationTimeout,
		MaxRetries:            -1,
		ContextTimeoutEnabled: true,
	})
	t.Cleanup(client.Close)
	return client, log
}

func serveRedisCommandTestConn(conn net.Conn, log *redisCommandLog, handler func([]string) string) {
	defer func() {
		_ = conn.Close()
	}()
	reader := bufio.NewReader(conn)
	for {
		args, errRead := readRedisCommand(reader)
		if errRead != nil {
			return
		}
		log.Append(args)
		response := "+OK\r\n"
		if handler != nil {
			response = handler(args)
		}
		if _, errWrite := io.WriteString(conn, response); errWrite != nil {
			return
		}
	}
}

func readRedisCommand(reader *bufio.Reader) ([]string, error) {
	line, errRead := reader.ReadString('\n')
	if errRead != nil {
		return nil, errRead
	}
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "*") {
		return nil, fmt.Errorf("expected array, got %q", line)
	}
	count, errCount := strconv.Atoi(strings.TrimPrefix(line, "*"))
	if errCount != nil {
		return nil, errCount
	}
	args := make([]string, 0, count)
	for i := 0; i < count; i++ {
		bulkLine, errBulk := reader.ReadString('\n')
		if errBulk != nil {
			return nil, errBulk
		}
		bulkLine = strings.TrimSpace(bulkLine)
		if !strings.HasPrefix(bulkLine, "$") {
			return nil, fmt.Errorf("expected bulk string, got %q", bulkLine)
		}
		size, errSize := strconv.Atoi(strings.TrimPrefix(bulkLine, "$"))
		if errSize != nil {
			return nil, errSize
		}
		payload := make([]byte, size+2)
		if _, errFull := io.ReadFull(reader, payload); errFull != nil {
			return nil, errFull
		}
		args = append(args, string(payload[:size]))
	}
	return args, nil
}
