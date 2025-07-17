package mocks_test

import (
	"context"
	"errors"
	"testing"

	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockLabelManagerInterface(t *testing.T) {
	ctx := context.Background()

	t.Run("基本的な動作", func(t *testing.T) {
		manager := mocks.NewMockLabelManagerInterface()

		// TransitionLabelWithRetry のモック設定
		manager.On("TransitionLabelWithRetry", ctx, "owner", "repo", 123).
			Return(true, nil)

		// 実行
		transitioned, err := manager.TransitionLabelWithRetry(ctx, "owner", "repo", 123)
		require.NoError(t, err)
		assert.True(t, transitioned)

		manager.AssertExpectations(t)
	})

	t.Run("TransitionLabelWithInfoWithRetry", func(t *testing.T) {
		manager := mocks.NewMockLabelManagerInterface()

		expectedInfo := &github.TransitionInfo{
			TransitionFound: true,
			FromLabel:       "status:ready",
			ToLabel:         "status:in-progress",
			CurrentLabels:   []string{"status:in-progress", "priority:high"},
		}

		manager.On("TransitionLabelWithInfoWithRetry", ctx, "owner", "repo", 456).
			Return(true, expectedInfo, nil)

		// 実行
		transitioned, info, err := manager.TransitionLabelWithInfoWithRetry(ctx, "owner", "repo", 456)
		require.NoError(t, err)
		assert.True(t, transitioned)
		assert.Equal(t, expectedInfo, info)

		manager.AssertExpectations(t)
	})

	t.Run("EnsureLabelsExistWithRetry", func(t *testing.T) {
		manager := mocks.NewMockLabelManagerInterface()

		manager.On("EnsureLabelsExistWithRetry", ctx, "owner", "repo").
			Return(nil)

		// 実行
		err := manager.EnsureLabelsExistWithRetry(ctx, "owner", "repo")
		assert.NoError(t, err)

		manager.AssertExpectations(t)
	})

	t.Run("デフォルト動作の確認", func(t *testing.T) {
		manager := mocks.NewMockLabelManagerInterface().WithDefaultBehavior()

		t.Run("TransitionLabelWithRetry", func(t *testing.T) {
			transitioned, err := manager.TransitionLabelWithRetry(ctx, "any", "repo", 999)
			assert.NoError(t, err)
			assert.False(t, transitioned)
		})

		t.Run("TransitionLabelWithInfoWithRetry", func(t *testing.T) {
			transitioned, info, err := manager.TransitionLabelWithInfoWithRetry(ctx, "any", "repo", 999)
			assert.NoError(t, err)
			assert.False(t, transitioned)
			assert.NotNil(t, info)
			assert.False(t, info.TransitionFound)
		})

		t.Run("EnsureLabelsExistWithRetry", func(t *testing.T) {
			err := manager.EnsureLabelsExistWithRetry(ctx, "any", "repo")
			assert.NoError(t, err)
		})
	})

	t.Run("WithSuccessfulTransition ヘルパー", func(t *testing.T) {
		manager := mocks.NewMockLabelManagerInterface().
			WithSuccessfulTransition("owner", "repo", 123, "status:ready", "status:in-progress")

		// TransitionLabelWithRetry
		transitioned, err := manager.TransitionLabelWithRetry(ctx, "owner", "repo", 123)
		assert.NoError(t, err)
		assert.True(t, transitioned)

		// TransitionLabelWithInfoWithRetry
		transitioned2, info, err := manager.TransitionLabelWithInfoWithRetry(ctx, "owner", "repo", 123)
		assert.NoError(t, err)
		assert.True(t, transitioned2)
		assert.True(t, info.TransitionFound)
		assert.Equal(t, "status:ready", info.FromLabel)
		assert.Equal(t, "status:in-progress", info.ToLabel)
	})

	t.Run("WithTransitionError ヘルパー", func(t *testing.T) {
		expectedErr := errors.New("transition failed")
		manager := mocks.NewMockLabelManagerInterface().
			WithTransitionError("owner", "repo", 456, expectedErr)

		// TransitionLabelWithRetry
		transitioned, err := manager.TransitionLabelWithRetry(ctx, "owner", "repo", 456)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.False(t, transitioned)

		// TransitionLabelWithInfoWithRetry
		transitioned2, info, err := manager.TransitionLabelWithInfoWithRetry(ctx, "owner", "repo", 456)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.False(t, transitioned2)
		assert.Nil(t, info)
	})

	t.Run("WithLabelsEnsured ヘルパー", func(t *testing.T) {
		manager := mocks.NewMockLabelManagerInterface().
			WithLabelsEnsured("owner", "repo")

		err := manager.EnsureLabelsExistWithRetry(ctx, "owner", "repo")
		assert.NoError(t, err)
	})

	t.Run("WithLabelsEnsureError ヘルパー", func(t *testing.T) {
		expectedErr := errors.New("labels creation failed")
		manager := mocks.NewMockLabelManagerInterface().
			WithLabelsEnsureError("owner", "repo", expectedErr)

		err := manager.EnsureLabelsExistWithRetry(ctx, "owner", "repo")
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}
