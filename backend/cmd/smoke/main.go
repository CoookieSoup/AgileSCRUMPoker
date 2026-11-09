package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/api/generated"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/model"
)

const (
	defaultBaseURL = "http://localhost:8090"
	requestTimeout = 10 * time.Second
	eventTimeout   = 3 * time.Second
	messageBuffer  = 128
)

var errNotReady = errors.New("first websocket message was not ready")

type reply struct {
	status int
	body   []byte
}

type runner struct {
	baseURL string
	client  *http.Client
	failed  int
}

func main() {
	baseURL := strings.TrimRight(os.Getenv("BASE_URL"), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	fmt.Printf("smoke testing %s\n\n", baseURL)

	check := &runner{baseURL: baseURL, client: &http.Client{Timeout: requestTimeout}}
	check.run()

	if check.failed > 0 {
		fmt.Printf("\n%d check(s) failed\n", check.failed)
		os.Exit(1)
	}

	fmt.Println("\nall checks passed")
}

func (check *runner) run() {
	healthy := check.status("health check returns 200", check.call(http.MethodGet, "/healthz", "", nil),
		http.StatusOK)
	if !healthy {
		return
	}

	check.failure("create rejects a blank name",
		check.call(http.MethodPost, "/rooms", "", map[string]string{"displayName": "  "}),
		http.StatusUnprocessableEntity, "invalid_display_name")

	host, created := check.session("create room", "/rooms", "Smoke host")
	if !created {
		return
	}

	code := host.Room.Code
	roomPath := "/rooms/" + code

	check.failure("join rejects a malformed code",
		check.call(http.MethodPost, "/rooms/NOPE/participants", "", map[string]string{"displayName": "Guest"}),
		http.StatusNotFound, "room_not_found")

	guest, joined := check.session("join room", roomPath+"/participants", "Smoke guest")
	if !joined {
		return
	}

	messages, stopListening, err := check.listen(code, guest.SessionToken)
	check.expect("websocket authenticates", err == nil, fmt.Sprint(err))

	if err == nil {
		defer stopListening()
	}

	check.failure("bad token is unauthorized", check.call(http.MethodGet, roomPath, "not-a-token", nil),
		http.StatusUnauthorized, "unauthorized")

	state := check.state("guest reads room state", check.call(http.MethodGet, roomPath, guest.SessionToken, nil),
		http.StatusOK)
	check.expect("room lists both participants", len(state.Participants) == 2,
		fmt.Sprintf("%d participants", len(state.Participants)))

	check.voting(roomPath, host, guest)
	check.capacity(roomPath)

	check.status("guest leaves", check.call(http.MethodDelete, roomPath+"/participants/me", guest.SessionToken, nil),
		http.StatusNoContent)
	check.status("host closes room", check.call(http.MethodDelete, roomPath, host.SessionToken, nil),
		http.StatusNoContent)
	check.failure("closed room cannot be joined",
		check.call(http.MethodPost, roomPath+"/participants", "", map[string]string{"displayName": "Late"}),
		http.StatusNotFound, "room_not_found")

	if err == nil {
		check.events(messages)
	}
}

func (check *runner) voting(roomPath string, host generated.Session, guest generated.Session) {
	round := map[string]string{"label": "Smoke story", "jiraKey": "SMOKE-1"}

	check.failure("guest cannot start a round", check.call(http.MethodPost, roomPath+"/round", guest.SessionToken, round),
		http.StatusForbidden, "not_host")
	check.failure("invalid jira key is rejected",
		check.call(http.MethodPost, roomPath+"/round", host.SessionToken, map[string]string{"jiraKey": "nope"}),
		http.StatusUnprocessableEntity, "invalid_jira_key")
	check.state("host starts round", check.call(http.MethodPost, roomPath+"/round", host.SessionToken, round),
		http.StatusCreated)

	votePath := roomPath + "/round/vote"

	check.failure("card outside the deck is rejected",
		check.call(http.MethodPut, votePath, guest.SessionToken, map[string]string{"card": "4"}),
		http.StatusUnprocessableEntity, "invalid_card")
	check.status("guest votes", check.call(http.MethodPut, votePath, guest.SessionToken, map[string]string{"card": "5"}),
		http.StatusNoContent)
	check.status("guest changes vote",
		check.call(http.MethodPut, votePath, guest.SessionToken, map[string]string{"card": "8"}), http.StatusNoContent)
	check.status("host votes", check.call(http.MethodPut, votePath, host.SessionToken, map[string]string{"card": "3"}),
		http.StatusNoContent)

	hidden := check.state("guest reads hidden votes", check.call(http.MethodGet, roomPath, guest.SessionToken, nil),
		http.StatusOK)
	check.expect("votes stay hidden before reveal", hidden.Round != nil && len(hidden.Round.Votes) == 0,
		"votes visible before reveal")
	check.expect("guest sees own vote", hidden.OwnVote != nil && *hidden.OwnVote == "8", "own vote missing")

	check.failure("guest cannot reveal", check.call(http.MethodPost, roomPath+"/round/reveal", guest.SessionToken, nil),
		http.StatusForbidden, "not_host")

	revealed := check.state("host reveals votes",
		check.call(http.MethodPost, roomPath+"/round/reveal", host.SessionToken, nil), http.StatusOK)
	check.expect("reveal shows both votes", revealed.Round != nil && len(revealed.Round.Votes) == 2,
		"expected two revealed votes")

	check.failure("vote after reveal is rejected",
		check.call(http.MethodPut, votePath, guest.SessionToken, map[string]string{"card": "1"}),
		http.StatusConflict, "round_revealed")
}

func (check *runner) capacity(roomPath string) {
	filled := true

	for index := 3; index <= model.MaxParticipants; index++ {
		filling := check.call(http.MethodPost, roomPath+"/participants", "",
			map[string]string{"displayName": fmt.Sprintf("Filler %d", index)})
		filled = filled && filling.status == http.StatusCreated
	}

	check.expect("room fills to 20 participants", filled, "a filler join failed")
	check.failure("21st participant is rejected",
		check.call(http.MethodPost, roomPath+"/participants", "", map[string]string{"displayName": "One too many"}),
		http.StatusConflict, "room_full")
}

func (check *runner) events(messages <-chan string) {
	received := []string{}
	deadline := time.After(eventTimeout)

	for collecting := true; collecting; {
		select {
		case message, open := <-messages:
			if !open {
				collecting = false

				continue
			}

			received = append(received, message)
			collecting = !strings.Contains(message, `"room_closed"`)
		case <-deadline:
			collecting = false
		}
	}

	revealedAt := -1

	for index, message := range received {
		if strings.Contains(message, `"votes_revealed"`) {
			revealedAt = index

			break
		}
	}

	leaked := false

	for index, message := range received {
		fmt.Printf("      event  %s\n", message)

		if (revealedAt < 0 || index < revealedAt) && strings.Contains(message, `"card"`) {
			leaked = true
		}
	}

	check.expect("no card is streamed before reveal", !leaked, "a card appeared before votes_revealed")
	check.expect("votes_revealed is streamed", revealedAt >= 0, "event missing")
	check.expect("room_closed is streamed", len(received) > 0 &&
		strings.Contains(received[len(received)-1], `"room_closed"`), "event missing")
}

func (check *runner) listen(code string, token string) (<-chan string, func(), error) {
	ctx, cancel := context.WithCancel(context.Background())
	address := "ws" + strings.TrimPrefix(check.baseURL, "http") + "/api/v1/rooms/" + code + "/events"

	connection, _, err := websocket.Dial(ctx, address, nil)
	if err != nil {
		cancel()

		return nil, nil, fmt.Errorf("dial: %w", err)
	}

	err = wsjson.Write(ctx, connection, map[string]string{"type": "auth", "token": token})
	if err != nil {
		cancel()

		return nil, nil, fmt.Errorf("send auth: %w", err)
	}

	var ready map[string]any

	err = wsjson.Read(ctx, connection, &ready)
	if err != nil {
		cancel()

		return nil, nil, fmt.Errorf("read ready message: %w", err)
	}

	if ready["type"] != "ready" {
		cancel()

		return nil, nil, errNotReady
	}

	messages := make(chan string, messageBuffer)

	go func() {
		defer close(messages)

		for {
			_, content, readErr := connection.Read(ctx)
			if readErr != nil {
				return
			}

			messages <- strings.TrimSpace(string(content))
		}
	}()

	stop := func() {
		cancel()

		_ = connection.CloseNow()
	}

	return messages, stop, nil
}

func (check *runner) session(name string, path string, displayName string) (generated.Session, bool) {
	result := check.call(http.MethodPost, path, "", map[string]string{"displayName": displayName})
	if !check.status(name, result, http.StatusCreated) {
		return generated.Session{}, false
	}

	var session generated.Session

	err := json.Unmarshal(result.body, &session)
	check.expect(name+" returns a session", err == nil && session.SessionToken != "", fmt.Sprint(err))

	return session, err == nil
}

func (check *runner) state(name string, result reply, status int) generated.RoomState {
	var state generated.RoomState

	if !check.status(name, result, status) {
		return state
	}

	err := json.Unmarshal(result.body, &state)
	if err != nil {
		check.expect(name+" returns room state", false, err.Error())
	}

	return state
}

func (check *runner) failure(name string, result reply, status int, code string) {
	var body generated.Error

	_ = json.Unmarshal(result.body, &body)

	check.expect(name, result.status == status && body.Code == code,
		fmt.Sprintf("want %d %s, got %d %s", status, code, result.status, strings.TrimSpace(string(result.body))))
}

func (check *runner) status(name string, result reply, status int) bool {
	passed := result.status == status
	check.expect(name, passed,
		fmt.Sprintf("want %d, got %d %s", status, result.status, strings.TrimSpace(string(result.body))))

	return passed
}

func (check *runner) expect(name string, passed bool, detail string) {
	if passed {
		fmt.Printf("PASS  %s\n", name)

		return
	}

	check.failed++

	fmt.Printf("FAIL  %s: %s\n", name, detail)
}

func (check *runner) call(method string, path string, token string, body any) reply {
	var payload io.Reader = http.NoBody

	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return reply{body: []byte(err.Error())}
		}

		payload = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(context.Background(), method, check.baseURL+"/api/v1"+path, payload)
	if err != nil {
		return reply{body: []byte(err.Error())}
	}

	request.Header.Set("Content-Type", "application/json")

	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := check.client.Do(request)
	if err != nil {
		return reply{body: []byte(err.Error())}
	}

	defer func() {
		_ = response.Body.Close()
	}()

	content, err := io.ReadAll(response.Body)
	if err != nil {
		return reply{status: response.StatusCode, body: []byte(err.Error())}
	}

	return reply{status: response.StatusCode, body: content}
}
