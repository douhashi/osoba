package gh

import (
	"context"
	"errors"
)

// CheckInstalled はghコマンドがインストールされているかチェックする
func CheckInstalled(ctx context.Context, executor CommandExecutor) (bool, error) {
	_, err := executor.Execute(ctx, "gh", "--version")
	if err != nil {
		var execErr *ExecError
		if errors.As(err, &execErr) {
			// コマンドが見つからない場合は、インストールされていないと判断
			return false, nil
		}
		// それ以外のエラーは予期しないエラーとして返す
		return false, err
	}
	return true, nil
}

// CheckAuth はghコマンドが認証済みかチェックする
func CheckAuth(ctx context.Context, executor CommandExecutor) (bool, error) {
	_, err := executor.Execute(ctx, "gh", "auth", "status")
	if err != nil {
		var execErr *ExecError
		if errors.As(err, &execErr) {
			// ExitCode 1 は未認証を示す
			if execErr.ExitCode == 1 {
				return false, nil
			}
		}
		// それ以外のエラーは予期しないエラーとして返す
		return false, err
	}
	return true, nil
}
