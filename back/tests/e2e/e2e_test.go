package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

type e2eConfig struct {
	APIBaseURL        string
	RequestTimeout    time.Duration
	EventuallyTimeout time.Duration
	EventuallyTick    time.Duration
}

func loadE2EConfig() e2eConfig {
	return e2eConfig{
		APIBaseURL:        "http://localhost:8080",
		RequestTimeout:    10 * time.Second,
		EventuallyTimeout: 45 * time.Second,
		EventuallyTick:    500 * time.Millisecond,
	}
}

type e2eClient struct {
	cfg          e2eConfig
	httpClient   *http.Client
	streamClient *http.Client
}

func newE2EClient(cfg e2eConfig) *e2eClient {
	return &e2eClient{
		cfg:          cfg,
		httpClient:   &http.Client{Timeout: cfg.RequestTimeout},
		streamClient: &http.Client{},
	}
}

func (c *e2eClient) requireServicesUp() error {
	if err := c.checkHealth(c.cfg.APIBaseURL + "/healthz"); err != nil {
		return fmt.Errorf("api-service healthcheck failed: %w", err)
	}
	return nil
}

func (c *e2eClient) checkHealth(target string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.cfg.RequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	return nil
}

func (c *e2eClient) doJSON(method, baseURL, path, accessToken string, body any) (int, map[string]any, []byte, error) {
	target := strings.TrimRight(baseURL, "/") + path

	var payload io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, nil, nil, err
		}
		payload = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, target, payload)
	if err != nil {
		return 0, nil, nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, nil, err
	}

	obj := map[string]any{}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed != "" {
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		if err := dec.Decode(&obj); err != nil {
			obj = nil
		}
	}

	return resp.StatusCode, obj, raw, nil
}

func (c *e2eClient) mustJSON(t *testing.T, method, baseURL, path, accessToken string, body any, wantStatus int) map[string]any {
	t.Helper()

	status, obj, raw, err := c.doJSON(method, baseURL, path, accessToken, body)
	if err != nil {
		t.Fatalf("%s %s request error: %v", method, path, err)
	}
	if status != wantStatus {
		t.Fatalf("%s %s status=%d want=%d body=%s", method, path, status, wantStatus, renderErrorBody(obj, raw))
	}

	if obj == nil {
		return map[string]any{}
	}
	return obj
}

func (c *e2eClient) eventually(t *testing.T, title string, fn func() (bool, string, error)) {
	t.Helper()

	deadline := time.Now().Add(c.cfg.EventuallyTimeout)
	lastMessage := "condition is not met yet"

	for {
		ok, msg, err := fn()
		if err != nil {
			lastMessage = err.Error()
		} else if strings.TrimSpace(msg) != "" {
			lastMessage = msg
		}

		if ok {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for %s: %s", title, lastMessage)
		}

		time.Sleep(c.cfg.EventuallyTick)
	}
}

func TestBackendE2EFullFlow(t *testing.T) {
	cfg := loadE2EConfig()
	client := newE2EClient(cfg)

	if err := client.requireServicesUp(); err != nil {
		t.Skipf("e2e skipped: %v", err)
	}

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	password := "StrongPass123!"
	user1Email := fmt.Sprintf("e2e-user1-%s@example.com", suffix)
	user2Email := fmt.Sprintf("e2e-user2-%s@example.com", suffix)
	tagPrimary := fmt.Sprintf("e2e-primary-%s", suffix)
	tagSecondary := fmt.Sprintf("e2e-secondary-%s", suffix)
	question := fmt.Sprintf("E2E poll question %s", suffix)
	updatedQuestion := question + " updated"

	registerBody1 := map[string]any{
		"email":     user1Email,
		"password":  password,
		"country":   "RU",
		"gender":    "male",
		"birthYear": 1998,
	}
	registerBody2 := map[string]any{
		"email":     user2Email,
		"password":  password,
		"country":   "DE",
		"gender":    "female",
		"birthYear": 2000,
	}

	registerResp1 := client.mustJSON(t, http.MethodPost, cfg.APIBaseURL, "/v1/auth/register", "", registerBody1, http.StatusOK)
	accessToken1 := mustStringPath(t, registerResp1, "tokens", "accessToken")
	refreshToken1 := mustStringPath(t, registerResp1, "tokens", "refreshToken")
	if refreshToken1 == "" {
		t.Fatalf("register response must contain refreshToken")
	}

	meResp := client.mustJSON(t, http.MethodGet, cfg.APIBaseURL, "/v1/auth/me", accessToken1, nil, http.StatusOK)
	user1ID := mustStringPath(t, meResp, "user", "id")
	if gotEmail := strings.ToLower(mustStringPath(t, meResp, "user", "email")); gotEmail != user1Email {
		t.Fatalf("unexpected /v1/auth/me email=%s want=%s", gotEmail, user1Email)
	}

	updatedCountry := "US"
	updatedGender := "other"
	updatedBirthYear := int32(1997)
	updateMeResp := client.mustJSON(t, http.MethodPatch, cfg.APIBaseURL, "/v1/auth/me", accessToken1, map[string]any{
		"country":   updatedCountry,
		"gender":    updatedGender,
		"birthYear": updatedBirthYear,
	}, http.StatusOK)

	if gotCountry := mustStringPath(t, updateMeResp, "user", "country"); gotCountry != updatedCountry {
		t.Fatalf("unexpected updated country=%s want=%s", gotCountry, updatedCountry)
	}
	if gotGender := mustStringPath(t, updateMeResp, "user", "gender"); gotGender != updatedGender {
		t.Fatalf("unexpected updated gender=%s want=%s", gotGender, updatedGender)
	}
	if gotBirthYear := mustInt32Path(t, updateMeResp, "user", "birthYear"); gotBirthYear != updatedBirthYear {
		t.Fatalf("unexpected updated birthYear=%d want=%d", gotBirthYear, updatedBirthYear)
	}

	loginResp := client.mustJSON(t, http.MethodPost, cfg.APIBaseURL, "/v1/auth/login", "", map[string]any{
		"email":    user1Email,
		"password": password,
	}, http.StatusOK)
	loginRefreshToken := mustStringPath(t, loginResp, "tokens", "refreshToken")

	refreshResp := client.mustJSON(t, http.MethodPost, cfg.APIBaseURL, "/v1/auth/refresh", "", map[string]any{
		"refreshToken": loginRefreshToken,
	}, http.StatusOK)
	accessToken1 = mustStringPath(t, refreshResp, "tokens", "accessToken")
	activeRefreshToken := mustStringPath(t, refreshResp, "tokens", "refreshToken")

	registerResp2 := client.mustJSON(t, http.MethodPost, cfg.APIBaseURL, "/v1/auth/register", "", registerBody2, http.StatusOK)
	accessToken2 := mustStringPath(t, registerResp2, "tokens", "accessToken")

	client.mustJSON(t, http.MethodPost, cfg.APIBaseURL, "/v1/tags", accessToken1, map[string]any{"name": tagPrimary}, http.StatusOK)
	client.mustJSON(t, http.MethodPost, cfg.APIBaseURL, "/v1/tags", accessToken1, map[string]any{"name": tagSecondary}, http.StatusOK)

	listTagsResp := client.mustJSON(t, http.MethodGet, cfg.APIBaseURL, "/v1/tags", "", nil, http.StatusOK)
	tagItems := mustArrayPath(t, listTagsResp, "items")
	if !containsItemByStringField(tagItems, "name", tagPrimary) {
		t.Fatalf("tag %s was not found in /v1/tags", tagPrimary)
	}
	if !containsItemByStringField(tagItems, "name", tagSecondary) {
		t.Fatalf("tag %s was not found in /v1/tags", tagSecondary)
	}

	createPollResp := client.mustJSON(t, http.MethodPost, cfg.APIBaseURL, "/v1/polls", accessToken1, map[string]any{
		"question":    question,
		"type":        "POLL_TYPE_SINGLE_CHOICE",
		"isAnonymous": true,
		"endsAt":      time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
		"options":     []string{"Option A", "Option B", "Option C"},
		"tags":        []string{tagPrimary, tagSecondary},
	}, http.StatusOK)

	pollID := mustStringPath(t, createPollResp, "poll", "id")
	creatorID := mustStringPath(t, createPollResp, "poll", "creatorId")
	if creatorID != user1ID {
		t.Fatalf("unexpected poll creatorId=%s want=%s", creatorID, user1ID)
	}

	pollOptions := mustArrayPath(t, createPollResp, "poll", "options")
	if len(pollOptions) < 2 {
		t.Fatalf("expected at least 2 poll options, got %d", len(pollOptions))
	}

	optionAID := mustStringPath(t, mustObject(t, pollOptions[0], "poll.options[0]"), "id")
	optionBID := mustStringPath(t, mustObject(t, pollOptions[1], "poll.options[1]"), "id")

	getPollResp := client.mustJSON(t, http.MethodGet, cfg.APIBaseURL, "/v1/polls/"+url.PathEscape(pollID), "", nil, http.StatusOK)
	if gotID := mustStringPath(t, getPollResp, "poll", "id"); gotID != pollID {
		t.Fatalf("unexpected poll id from GET /v1/polls/{id}: %s", gotID)
	}

	listPollsQuery := url.Values{}
	listPollsQuery.Set("limit", "20")
	listPollsQuery.Add("tags", tagPrimary)
	listPollsResp := client.mustJSON(t, http.MethodGet, cfg.APIBaseURL, "/v1/polls?"+listPollsQuery.Encode(), "", nil, http.StatusOK)
	if _, ok := findItemByID(mustArrayPath(t, listPollsResp, "items"), pollID); !ok {
		t.Fatalf("created poll %s was not found in /v1/polls", pollID)
	}

	updatePollResp := client.mustJSON(t, http.MethodPatch, cfg.APIBaseURL, "/v1/polls/"+url.PathEscape(pollID), accessToken1, map[string]any{
		"question":    updatedQuestion,
		"isAnonymous": false,
		"tags":        []string{tagPrimary},
	}, http.StatusOK)
	if gotQuestion := mustStringPath(t, updatePollResp, "poll", "question"); gotQuestion != updatedQuestion {
		t.Fatalf("poll was not updated: question=%s want=%s", gotQuestion, updatedQuestion)
	}
	if gotAnon := mustBoolPath(t, updatePollResp, "poll", "isAnonymous"); gotAnon {
		t.Fatalf("poll isAnonymous should be false after update")
	}

	forbiddenStatus, forbiddenObj, forbiddenRaw, err := client.doJSON(http.MethodPatch, cfg.APIBaseURL, "/v1/polls/"+url.PathEscape(pollID), accessToken2, map[string]any{
		"question": "should-not-update",
	})
	if err != nil {
		t.Fatalf("forbidden poll update request error: %v", err)
	}
	if forbiddenStatus != http.StatusForbidden {
		t.Fatalf("expected 403 for non-owner poll update, got %d body=%s", forbiddenStatus, renderErrorBody(forbiddenObj, forbiddenRaw))
	}

	votePath := "/v1/polls/" + url.PathEscape(pollID) + "/vote"
	voteResp1 := client.mustJSON(t, http.MethodPost, cfg.APIBaseURL, votePath, accessToken1, map[string]any{
		"optionIds": []string{optionAID},
	}, http.StatusOK)
	if gotPollID := mustStringPath(t, voteResp1, "pollId"); gotPollID != pollID {
		t.Fatalf("unexpected pollId in vote response: %s", gotPollID)
	}

	getVoteResp1 := client.mustJSON(t, http.MethodGet, cfg.APIBaseURL, votePath, accessToken1, nil, http.StatusOK)
	if !mustBoolPath(t, getVoteResp1, "hasVoted") {
		t.Fatalf("expected hasVoted=true after first vote")
	}
	if !containsString(mustArrayPath(t, getVoteResp1, "optionIds"), optionAID) {
		t.Fatalf("expected option %s in current user vote", optionAID)
	}

	client.mustJSON(t, http.MethodPost, cfg.APIBaseURL, votePath, accessToken1, map[string]any{
		"optionIds": []string{optionBID},
	}, http.StatusOK)

	getVoteResp2 := client.mustJSON(t, http.MethodGet, cfg.APIBaseURL, votePath, accessToken1, nil, http.StatusOK)
	if !containsString(mustArrayPath(t, getVoteResp2, "optionIds"), optionBID) {
		t.Fatalf("expected option %s after vote update", optionBID)
	}

	client.mustJSON(t, http.MethodDelete, cfg.APIBaseURL, votePath, accessToken1, nil, http.StatusOK)
	getVoteResp3 := client.mustJSON(t, http.MethodGet, cfg.APIBaseURL, votePath, accessToken1, nil, http.StatusOK)
	if mustBoolPath(t, getVoteResp3, "hasVoted") {
		t.Fatalf("expected hasVoted=false after vote removal")
	}
	if len(mustArrayPath(t, getVoteResp3, "optionIds")) != 0 {
		t.Fatalf("expected empty optionIds after vote removal")
	}

	client.mustJSON(t, http.MethodPost, cfg.APIBaseURL, votePath, accessToken1, map[string]any{
		"optionIds": []string{optionAID},
	}, http.StatusOK)

	client.eventually(t, "feed contains created poll", func() (bool, string, error) {
		query := url.Values{}
		query.Set("limit", "20")
		query.Add("tags", tagPrimary)

		status, obj, raw, err := client.doJSON(http.MethodGet, cfg.APIBaseURL, "/v1/feed?"+query.Encode(), "", nil)
		if err != nil {
			return false, "", err
		}
		if status != http.StatusOK {
			return false, fmt.Sprintf("status=%d body=%s", status, renderErrorBody(obj, raw)), nil
		}
		if _, ok := findItemByID(mustArrayPath(t, obj, "items"), pollID); !ok {
			return false, "poll is not present in feed yet", nil
		}
		return true, "", nil
	})

	client.eventually(t, "trending contains created poll", func() (bool, string, error) {
		status, obj, raw, err := client.doJSON(http.MethodGet, cfg.APIBaseURL, "/v1/feed/trending?limit=20", "", nil)
		if err != nil {
			return false, "", err
		}
		if status != http.StatusOK {
			return false, fmt.Sprintf("status=%d body=%s", status, renderErrorBody(obj, raw)), nil
		}
		if _, ok := findItemByID(mustArrayPath(t, obj, "items"), pollID); !ok {
			return false, "poll is not present in trending yet", nil
		}
		return true, "", nil
	})

	client.eventually(t, "user polls contains created poll", func() (bool, string, error) {
		status, obj, raw, err := client.doJSON(http.MethodGet, cfg.APIBaseURL, "/v1/feed/user/"+url.PathEscape(user1ID)+"?limit=20", "", nil)
		if err != nil {
			return false, "", err
		}
		if status != http.StatusOK {
			return false, fmt.Sprintf("status=%d body=%s", status, renderErrorBody(obj, raw)), nil
		}
		if _, ok := findItemByID(mustArrayPath(t, obj, "items"), pollID); !ok {
			return false, "poll is not present in user feed yet", nil
		}
		return true, "", nil
	})

	client.eventually(t, "my polls contains created poll", func() (bool, string, error) {
		status, obj, raw, err := client.doJSON(http.MethodGet, cfg.APIBaseURL, "/v1/feed/me?limit=20", accessToken1, nil)
		if err != nil {
			return false, "", err
		}
		if status != http.StatusOK {
			return false, fmt.Sprintf("status=%d body=%s", status, renderErrorBody(obj, raw)), nil
		}
		if _, ok := findItemByID(mustArrayPath(t, obj, "items"), pollID); !ok {
			return false, "poll is not present in my polls yet", nil
		}
		return true, "", nil
	})

	client.eventually(t, "analytics summary is populated", func() (bool, string, error) {
		status, obj, raw, err := client.doJSON(http.MethodGet, cfg.APIBaseURL, "/v1/polls/"+url.PathEscape(pollID)+"/analytics", "", nil)
		if err != nil {
			return false, "", err
		}
		if status != http.StatusOK {
			return false, fmt.Sprintf("status=%d body=%s", status, renderErrorBody(obj, raw)), nil
		}
		totalVotes, err := int64ByPath(obj, "totalVotes")
		if err != nil {
			return false, fmt.Sprintf("totalVotes parse error: %v", err), nil
		}
		if totalVotes < 1 {
			return false, fmt.Sprintf("totalVotes=%d", totalVotes), nil
		}
		return true, "", nil
	})

	countryStatsResp := client.mustJSON(t, http.MethodGet, cfg.APIBaseURL, "/v1/polls/"+url.PathEscape(pollID)+"/analytics/countries", "", nil, http.StatusOK)
	if len(mustArrayPath(t, countryStatsResp, "items")) == 0 {
		t.Fatalf("expected non-empty analytics countries stats")
	}

	genderStatsResp := client.mustJSON(t, http.MethodGet, cfg.APIBaseURL, "/v1/polls/"+url.PathEscape(pollID)+"/analytics/gender", "", nil, http.StatusOK)
	if len(mustArrayPath(t, genderStatsResp, "items")) == 0 {
		t.Fatalf("expected non-empty analytics gender stats")
	}

	ageStatsResp := client.mustJSON(t, http.MethodGet, cfg.APIBaseURL, "/v1/polls/"+url.PathEscape(pollID)+"/analytics/age", "", nil, http.StatusOK)
	if len(mustArrayPath(t, ageStatsResp, "items")) == 0 {
		t.Fatalf("expected non-empty analytics age stats")
	}

	deletePollResp := client.mustJSON(t, http.MethodDelete, cfg.APIBaseURL, "/v1/polls/"+url.PathEscape(pollID), accessToken1, nil, http.StatusOK)
	if !mustBoolPath(t, deletePollResp, "success") {
		t.Fatalf("poll delete response must have success=true")
	}

	statusAfterDelete, objAfterDelete, rawAfterDelete, err := client.doJSON(http.MethodGet, cfg.APIBaseURL, "/v1/polls/"+url.PathEscape(pollID), "", nil)
	if err != nil {
		t.Fatalf("get poll after delete request error: %v", err)
	}
	if statusAfterDelete != http.StatusNotFound {
		t.Fatalf("expected 404 after poll delete, got %d body=%s", statusAfterDelete, renderErrorBody(objAfterDelete, rawAfterDelete))
	}

	client.eventually(t, "feed no longer contains deleted poll", func() (bool, string, error) {
		query := url.Values{}
		query.Set("limit", "20")
		query.Add("tags", tagPrimary)

		status, obj, raw, err := client.doJSON(http.MethodGet, cfg.APIBaseURL, "/v1/feed?"+query.Encode(), "", nil)
		if err != nil {
			return false, "", err
		}
		if status != http.StatusOK {
			return false, fmt.Sprintf("status=%d body=%s", status, renderErrorBody(obj, raw)), nil
		}
		if _, ok := findItemByID(mustArrayPath(t, obj, "items"), pollID); ok {
			return false, "poll is still present in feed", nil
		}
		return true, "", nil
	})

	client.mustJSON(t, http.MethodPost, cfg.APIBaseURL, "/v1/auth/logout", accessToken1, map[string]any{
		"refreshToken": activeRefreshToken,
	}, http.StatusOK)

	refreshAfterLogoutStatus, refreshAfterLogoutObj, refreshAfterLogoutRaw, err := client.doJSON(http.MethodPost, cfg.APIBaseURL, "/v1/auth/refresh", "", map[string]any{
		"refreshToken": activeRefreshToken,
	})
	if err != nil {
		t.Fatalf("refresh after logout request error: %v", err)
	}
	if refreshAfterLogoutStatus == http.StatusOK {
		t.Fatalf("refresh token must be revoked after logout, got status=200 body=%s", renderErrorBody(refreshAfterLogoutObj, refreshAfterLogoutRaw))
	}

	logoutAllResp := client.mustJSON(t, http.MethodPost, cfg.APIBaseURL, "/v1/auth/logout-all", accessToken1, map[string]any{}, http.StatusOK)
	if !mustBoolPath(t, logoutAllResp, "success") {
		t.Fatalf("logout-all response must have success=true")
	}
}

func renderErrorBody(obj map[string]any, raw []byte) string {
	if obj != nil {
		if msg, ok := obj["message"].(string); ok && strings.TrimSpace(msg) != "" {
			return msg
		}
	}
	return strings.TrimSpace(string(raw))
}

func lookupPath(root map[string]any, path ...string) (any, bool) {
	if len(path) == 0 {
		return nil, false
	}

	var current any = root
	for _, key := range path {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := obj[key]
		if !ok {
			return nil, false
		}
		current = next
	}

	return current, true
}

func mustObject(t *testing.T, value any, field string) map[string]any {
	t.Helper()
	obj, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("field %s is not an object: %#v", field, value)
	}
	return obj
}

func mustArrayPath(t *testing.T, root map[string]any, path ...string) []any {
	t.Helper()
	value, ok := lookupPath(root, path...)
	if !ok {
		t.Fatalf("field %s is missing", strings.Join(path, "."))
	}
	arr, ok := value.([]any)
	if !ok {
		t.Fatalf("field %s is not an array: %#v", strings.Join(path, "."), value)
	}
	return arr
}

func mustStringPath(t *testing.T, root map[string]any, path ...string) string {
	t.Helper()
	value, ok := lookupPath(root, path...)
	if !ok {
		t.Fatalf("field %s is missing", strings.Join(path, "."))
	}
	s, ok := value.(string)
	if !ok {
		t.Fatalf("field %s is not a string: %#v", strings.Join(path, "."), value)
	}
	return s
}

func mustBoolPath(t *testing.T, root map[string]any, path ...string) bool {
	t.Helper()
	value, ok := lookupPath(root, path...)
	if !ok {
		t.Fatalf("field %s is missing", strings.Join(path, "."))
	}
	b, ok := value.(bool)
	if !ok {
		t.Fatalf("field %s is not a bool: %#v", strings.Join(path, "."), value)
	}
	return b
}

func mustInt32Path(t *testing.T, root map[string]any, path ...string) int32 {
	t.Helper()
	v, err := int64ByPath(root, path...)
	if err != nil {
		t.Fatalf("field %s is not an int32: %v", strings.Join(path, "."), err)
	}
	return int32(v)
}

func int64ByPath(root map[string]any, path ...string) (int64, error) {
	value, ok := lookupPath(root, path...)
	if !ok {
		return 0, fmt.Errorf("field %s is missing", strings.Join(path, "."))
	}
	return toInt64(value)
}

func toInt64(value any) (int64, error) {
	switch v := value.(type) {
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i, nil
		}
		if f, err := strconv.ParseFloat(v.String(), 64); err == nil {
			return int64(f), nil
		}
		return 0, fmt.Errorf("cannot parse json number %q", v.String())
	case string:
		if strings.TrimSpace(v) == "" {
			return 0, fmt.Errorf("empty numeric string")
		}
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, err
		}
		return i, nil
	case float64:
		return int64(v), nil
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("unsupported numeric type %T", value)
	}
}

func containsItemByStringField(items []any, field, want string) bool {
	for _, raw := range items {
		obj, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		value, ok := obj[field].(string)
		if ok && value == want {
			return true
		}
	}
	return false
}

func containsString(items []any, want string) bool {
	for _, raw := range items {
		s, ok := raw.(string)
		if ok && s == want {
			return true
		}
	}
	return false
}

func findItemByID(items []any, wantID string) (map[string]any, bool) {
	for _, raw := range items {
		obj, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, _ := obj["id"].(string)
		if id == wantID {
			return obj, true
		}
	}
	return nil, false
}

func stringByAnyKey(obj map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		v, ok := obj[key]
		if !ok {
			continue
		}
		s, ok := v.(string)
		if !ok {
			continue
		}
		return s, true
	}
	return "", false
}
