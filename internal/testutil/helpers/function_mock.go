package helpers

import (
	"reflect"
	"sync"
)

// グローバルなモッカーリストを管理
var (
	allMockers []*FunctionMocker
	mockersMu  sync.Mutex
)

// FunctionMocker は関数変数をモックするためのヘルパー
type FunctionMocker struct {
	restoreFuncs []func()
}

// NewFunctionMocker は新しいFunctionMockerを作成
func NewFunctionMocker() *FunctionMocker {
	mocker := &FunctionMocker{
		restoreFuncs: make([]func(), 0),
	}

	// グローバルリストに追加
	mockersMu.Lock()
	allMockers = append(allMockers, mocker)
	mockersMu.Unlock()

	return mocker
}

// MockFunc は関数変数をモックする
// funcPtr は関数変数へのポインタ、mockImpl はモック実装
func (m *FunctionMocker) MockFunc(funcPtr interface{}, mockImpl interface{}) *FunctionMocker {
	// リフレクションを使用して関数変数を操作
	funcPtrValue := reflect.ValueOf(funcPtr)
	if funcPtrValue.Kind() != reflect.Ptr {
		panic("funcPtr must be a pointer to a function variable")
	}

	funcValue := funcPtrValue.Elem()
	mockValue := reflect.ValueOf(mockImpl)

	// 元の関数を保存（コピーが必要）
	originalFunc := reflect.New(funcValue.Type()).Elem()
	if funcValue.IsValid() && !funcValue.IsNil() {
		originalFunc.Set(funcValue)
	}

	// モック関数を設定
	funcValue.Set(mockValue)

	// 復元関数を追加
	m.restoreFuncs = append(m.restoreFuncs, func() {
		if originalFunc.IsValid() && !originalFunc.IsNil() {
			funcValue.Set(originalFunc)
		} else {
			// 元がnilだった場合はゼロ値に戻す
			funcValue.Set(reflect.Zero(funcValue.Type()))
		}
	})

	return m
}

// Restore はすべてのモックを元に戻す
func (m *FunctionMocker) Restore() {
	// 逆順で復元（後からモックしたものを先に戻す）
	for i := len(m.restoreFuncs) - 1; i >= 0; i-- {
		m.restoreFuncs[i]()
	}
	m.restoreFuncs = nil

	// グローバルリストから削除
	mockersMu.Lock()
	for i, mocker := range allMockers {
		if mocker == m {
			allMockers = append(allMockers[:i], allMockers[i+1:]...)
			break
		}
	}
	mockersMu.Unlock()
}

// RestoreAll はすべてのFunctionMockerを元に戻す
func RestoreAll() {
	mockersMu.Lock()
	mockers := make([]*FunctionMocker, len(allMockers))
	copy(mockers, allMockers)
	mockersMu.Unlock()

	for _, mocker := range mockers {
		mocker.Restore()
	}
}
