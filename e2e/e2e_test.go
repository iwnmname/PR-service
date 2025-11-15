package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const baseURL = "http://app_e2e:8080"

type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func waitForService(t *testing.T) {
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(baseURL + "/statistics")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			t.Log("Service is ready")
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		t.Logf("Waiting for service... (%d/%d)", i+1, maxRetries)
		time.Sleep(2 * time.Second)
	}
	t.Fatal("Service did not start in time")
}

func TestE2E(t *testing.T) {
	waitForService(t)

	t.Run("Complete Flow", func(t *testing.T) {
		testCreateTeam(t)
		testGetTeam(t)
		testCreatePR(t)
		testSetUserActive(t)
		testReassignReviewer(t)
		testMergePR(t)
		testGetUserReviews(t)
		testStatistics(t)
	})

	t.Run("Error Cases", func(t *testing.T) {
		testDuplicateTeam(t)
		testDuplicatePR(t)
		testNotFound(t)
		testReassignAfterMerge(t)
	})
}

func testCreateTeam(t *testing.T) {
	payload := map[string]interface{}{
		"team_name": "engineering",
		"members": []map[string]interface{}{
			{"user_id": "e1", "username": "Alice", "is_active": true},
			{"user_id": "e2", "username": "Bob", "is_active": true},
			{"user_id": "e3", "username": "Charlie", "is_active": true},
			{"user_id": "e4", "username": "Dave", "is_active": true},
		},
	}

	resp, body := makeRequest(t, "POST", "/team/add", payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	team := result["team"].(map[string]interface{})
	assert.Equal(t, "engineering", team["team_name"])
	assert.Len(t, team["members"], 4)
}

func testGetTeam(t *testing.T) {
	resp, body := makeRequest(t, "GET", "/team/get?team_name=engineering", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.Equal(t, "engineering", result["team_name"])
	assert.Len(t, result["members"], 4)
}

func testCreatePR(t *testing.T) {
	payload := map[string]interface{}{
		"pull_request_id":   "e2e-pr-001",
		"pull_request_name": "Add feature",
		"author_id":         "e1",
	}

	resp, body := makeRequest(t, "POST", "/pullRequest/create", payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	pr := result["pr"].(map[string]interface{})
	assert.Equal(t, "e2e-pr-001", pr["pull_request_id"])
	assert.Equal(t, "OPEN", pr["status"])

	reviewers := pr["assigned_reviewers"].([]interface{})
	assert.Len(t, reviewers, 2)

	for _, reviewer := range reviewers {
		assert.NotEqual(t, "e1", reviewer)
	}
}

func testSetUserActive(t *testing.T) {
	payload := map[string]interface{}{
		"user_id":   "e2",
		"is_active": false,
	}

	resp, body := makeRequest(t, "POST", "/users/setIsActive", payload)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	user := result["user"].(map[string]interface{})
	assert.Equal(t, "e2", user["user_id"])
	assert.False(t, user["is_active"].(bool))
}

func testReassignReviewer(t *testing.T) {
	payload := map[string]interface{}{
		"pull_request_id": "e2e-pr-001",
		"old_user_id":     "e2",
	}

	resp, body := makeRequest(t, "POST", "/pullRequest/reassign", payload)

	if resp.StatusCode == http.StatusOK {
		var result map[string]interface{}
		err := json.Unmarshal(body, &result)
		require.NoError(t, err)

		pr := result["pr"].(map[string]interface{})
		replacedBy := result["replaced_by"].(string)

		assert.NotEmpty(t, replacedBy)
		assert.NotEqual(t, "e2", replacedBy)
		assert.NotEqual(t, "e1", replacedBy)

		reviewers := pr["assigned_reviewers"].([]interface{})
		assert.NotContains(t, reviewers, "e2")
	} else if resp.StatusCode == http.StatusConflict {
		var errResp ErrorResponse
		err := json.Unmarshal(body, &errResp)
		require.NoError(t, err)
		assert.Contains(t, []string{"NOT_ASSIGNED", "NO_CANDIDATE"}, errResp.Error.Code)
	} else {
		t.Fatalf("Unexpected status code: %d", resp.StatusCode)
	}
}

func testMergePR(t *testing.T) {
	payload := map[string]interface{}{
		"pull_request_id": "e2e-pr-001",
	}

	resp, body := makeRequest(t, "POST", "/pullRequest/merge", payload)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	pr := result["pr"].(map[string]interface{})
	assert.Equal(t, "MERGED", pr["status"])
	assert.NotNil(t, pr["mergedAt"])

	resp2, body2 := makeRequest(t, "POST", "/pullRequest/merge", payload)
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	var result2 map[string]interface{}
	err = json.Unmarshal(body2, &result2)
	require.NoError(t, err)

	pr2 := result2["pr"].(map[string]interface{})
	assert.Equal(t, "MERGED", pr2["status"])
	assert.Equal(t, pr["mergedAt"], pr2["mergedAt"])
}

func testGetUserReviews(t *testing.T) {
	resp, body := makeRequest(t, "GET", "/users/getReview?user_id=e3", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.Equal(t, "e3", result["user_id"])
	prs := result["pull_requests"].([]interface{})
	assert.NotEmpty(t, prs)
}

func testStatistics(t *testing.T) {
	resp, body := makeRequest(t, "GET", "/statistics", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.Greater(t, int(result["total_users"].(float64)), 0)
	assert.Greater(t, int(result["total_teams"].(float64)), 0)
	assert.Greater(t, int(result["total_prs"].(float64)), 0)

	prsByStatus := result["prs_by_status"].([]interface{})
	assert.NotEmpty(t, prsByStatus)

	topReviewers := result["top_reviewers"].([]interface{})
	assert.NotEmpty(t, topReviewers)
}

func testDuplicateTeam(t *testing.T) {
	payload := map[string]interface{}{
		"team_name": "engineering",
		"members":   []map[string]interface{}{},
	}

	resp, body := makeRequest(t, "POST", "/team/add", payload)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var errResp ErrorResponse
	err := json.Unmarshal(body, &errResp)
	require.NoError(t, err)
	assert.Equal(t, "TEAM_EXISTS", errResp.Error.Code)
}

func testDuplicatePR(t *testing.T) {
	payload := map[string]interface{}{
		"pull_request_id":   "e2e-pr-001",
		"pull_request_name": "Duplicate",
		"author_id":         "e1",
	}

	resp, body := makeRequest(t, "POST", "/pullRequest/create", payload)
	require.Equal(t, http.StatusConflict, resp.StatusCode)

	var errResp ErrorResponse
	err := json.Unmarshal(body, &errResp)
	require.NoError(t, err)
	assert.Equal(t, "PR_EXISTS", errResp.Error.Code)
}

func testNotFound(t *testing.T) {
	resp, body := makeRequest(t, "GET", "/team/get?team_name=nonexistent", nil)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	var errResp ErrorResponse
	err := json.Unmarshal(body, &errResp)
	require.NoError(t, err)
	assert.Equal(t, "NOT_FOUND", errResp.Error.Code)
}

func testReassignAfterMerge(t *testing.T) {
	payload := map[string]interface{}{
		"pull_request_id": "e2e-pr-001",
		"old_user_id":     "e3",
	}

	resp, body := makeRequest(t, "POST", "/pullRequest/reassign", payload)
	require.Equal(t, http.StatusConflict, resp.StatusCode)

	var errResp ErrorResponse
	err := json.Unmarshal(body, &errResp)
	require.NoError(t, err)
	assert.Equal(t, "PR_MERGED", errResp.Error.Code)
}

func makeRequest(t *testing.T, method, path string, payload interface{}) (*http.Response, []byte) {
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		require.NoError(t, err)
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, baseURL+path, body)
	require.NoError(t, err)

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	t.Logf("%s %s -> %d", method, path, resp.StatusCode)

	return resp, respBody
}