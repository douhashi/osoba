package github

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueJSONMarshaling(t *testing.T) {
	t.Run("Issue型のJSONマーシャリング", func(t *testing.T) {
		issue := &Issue{
			Number: Int(123),
			Title:  String("Test Issue"),
			State:  String("open"),
			Body:   String("This is a test issue"),
			User: &User{
				Login: String("testuser"),
			},
			Labels: []*Label{
				{
					Name:  String("bug"),
					Color: String("d73a4a"),
				},
				{
					Name:  String("help wanted"),
					Color: String("008672"),
				},
			},
			CreatedAt: &time.Time{},
			UpdatedAt: &time.Time{},
		}

		data, err := json.Marshal(issue)
		require.NoError(t, err)

		var unmarshaled Issue
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, 123, *unmarshaled.Number)
		assert.Equal(t, "Test Issue", *unmarshaled.Title)
		assert.Equal(t, "open", *unmarshaled.State)
		assert.Equal(t, "This is a test issue", *unmarshaled.Body)
		assert.Equal(t, "testuser", *unmarshaled.User.Login)
		assert.Len(t, unmarshaled.Labels, 2)
		assert.Equal(t, "bug", *unmarshaled.Labels[0].Name)
		assert.Equal(t, "help wanted", *unmarshaled.Labels[1].Name)
	})

	t.Run("nilフィールドの処理", func(t *testing.T) {
		issue := &Issue{
			Number: Int(456),
			Title:  String("Minimal Issue"),
			// Body, User, Labels はnil
		}

		data, err := json.Marshal(issue)
		require.NoError(t, err)

		var unmarshaled Issue
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, 456, *unmarshaled.Number)
		assert.Equal(t, "Minimal Issue", *unmarshaled.Title)
		assert.Nil(t, unmarshaled.Body)
		assert.Nil(t, unmarshaled.User)
		assert.Nil(t, unmarshaled.Labels)
	})
}

func TestLabelJSONMarshaling(t *testing.T) {
	label := &Label{
		ID:    Int64(12345),
		Name:  String("bug"),
		Color: String("d73a4a"),
	}

	data, err := json.Marshal(label)
	require.NoError(t, err)

	var unmarshaled Label
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, int64(12345), *unmarshaled.ID)
	assert.Equal(t, "bug", *unmarshaled.Name)
	assert.Equal(t, "d73a4a", *unmarshaled.Color)
}

func TestRepositoryJSONMarshaling(t *testing.T) {
	repo := &Repository{
		ID:       Int64(789),
		Name:     String("test-repo"),
		FullName: String("owner/test-repo"),
		Owner: &User{
			Login: String("owner"),
		},
		Private: Bool(false),
		HTMLURL: String("https://github.com/owner/test-repo"),
	}

	data, err := json.Marshal(repo)
	require.NoError(t, err)

	var unmarshaled Repository
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, int64(789), *unmarshaled.ID)
	assert.Equal(t, "test-repo", *unmarshaled.Name)
	assert.Equal(t, "owner/test-repo", *unmarshaled.FullName)
	assert.Equal(t, "owner", *unmarshaled.Owner.Login)
	assert.Equal(t, false, *unmarshaled.Private)
	assert.Equal(t, "https://github.com/owner/test-repo", *unmarshaled.HTMLURL)
}

func TestHelperFunctions(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		s := "test"
		ptr := String(s)
		assert.NotNil(t, ptr)
		assert.Equal(t, s, *ptr)
	})

	t.Run("Int", func(t *testing.T) {
		i := 42
		ptr := Int(i)
		assert.NotNil(t, ptr)
		assert.Equal(t, i, *ptr)
	})

	t.Run("Int64", func(t *testing.T) {
		i := int64(12345)
		ptr := Int64(i)
		assert.NotNil(t, ptr)
		assert.Equal(t, i, *ptr)
	})

	t.Run("Bool", func(t *testing.T) {
		b := true
		ptr := Bool(b)
		assert.NotNil(t, ptr)
		assert.Equal(t, b, *ptr)
	})
}

func TestRateLimitsJSONMarshaling(t *testing.T) {
	rl := &RateLimits{
		Core: &RateLimit{
			Limit:     5000,
			Remaining: 4999,
			Reset:     time.Now().Add(time.Hour),
		},
		Search: &RateLimit{
			Limit:     30,
			Remaining: 25,
			Reset:     time.Now().Add(time.Minute * 30),
		},
	}

	data, err := json.Marshal(rl)
	require.NoError(t, err)

	var unmarshaled RateLimits
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, 5000, unmarshaled.Core.Limit)
	assert.Equal(t, 4999, unmarshaled.Core.Remaining)
	assert.Equal(t, 30, unmarshaled.Search.Limit)
	assert.Equal(t, 25, unmarshaled.Search.Remaining)
}

func TestErrorResponseJSONMarshaling(t *testing.T) {
	errResp := &ErrorResponse{
		Message: "Not Found",
		Errors: []Error{
			{
				Resource: "Issue",
				Field:    "number",
				Code:     "invalid",
			},
		},
	}

	data, err := json.Marshal(errResp)
	require.NoError(t, err)

	var unmarshaled ErrorResponse
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, "Not Found", unmarshaled.Message)
	assert.Len(t, unmarshaled.Errors, 1)
	assert.Equal(t, "Issue", unmarshaled.Errors[0].Resource)
	assert.Equal(t, "number", unmarshaled.Errors[0].Field)
	assert.Equal(t, "invalid", unmarshaled.Errors[0].Code)
}

func TestIssueCommentJSONMarshaling(t *testing.T) {
	comment := &IssueComment{
		ID:   Int64(999),
		Body: String("This is a comment"),
		User: &User{
			Login: String("commenter"),
		},
		CreatedAt: &time.Time{},
		UpdatedAt: &time.Time{},
	}

	data, err := json.Marshal(comment)
	require.NoError(t, err)

	var unmarshaled IssueComment
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, int64(999), *unmarshaled.ID)
	assert.Equal(t, "This is a comment", *unmarshaled.Body)
	assert.Equal(t, "commenter", *unmarshaled.User.Login)
}

func TestListOptionsDefaults(t *testing.T) {
	opts := &ListOptions{}

	// デフォルト値の確認
	assert.Equal(t, 0, opts.Page)
	assert.Equal(t, 0, opts.PerPage)
}

func TestIssueListByRepoOptionsDefaults(t *testing.T) {
	opts := &IssueListByRepoOptions{}

	// デフォルト値の確認
	assert.Equal(t, "", opts.State)
	assert.Equal(t, "", opts.Sort)
	assert.Equal(t, "", opts.Direction)
	assert.Nil(t, opts.Labels)
}
