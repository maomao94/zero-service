package antsx_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
	"zero-service/common/antsx"
)

func TestReactorAndPromise(t *testing.T) {
	reactor, err := antsx.NewReactor(5)
	if err != nil {
		t.Fatal(err)
	}
	defer reactor.Release()

	ctx := context.Background()

	// 提交成功任务，返回 int
	p1, err := antsx.Submit(ctx, reactor, "task1", func(ctx context.Context) (int, error) {
		time.Sleep(100 * time.Millisecond)
		return 42, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// 链式转换 int -> string
	p2 := antsx.Then(ctx, p1, func(val int) (string, error) {
		return fmt.Sprintf("value is %d", val), nil
	})

	// 捕获错误（正常不会触发）
	p2.Catch(func(err error) {
		t.Errorf("unexpected error: %v", err)
	})

	// 等待结果
	res, err := p2.Await(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if res != "value is 42" {
		t.Fatalf("unexpected result: %s", res)
	}

	t.Logf("Success chain result: %s", res)

	// 测试失败任务
	pFail, err := antsx.Submit(ctx, reactor, "failTask", func(ctx context.Context) (string, error) {
		return "", errors.New("intentional failure")
	})
	if err != nil {
		t.Fatal(err)
	}

	// 捕获失败错误
	caught := false
	pFail.Catch(func(err error) {
		caught = true
		if err == nil || err.Error() != "intentional failure" {
			t.Errorf("unexpected error in Catch: %v", err)
		}
	})

	_, err = pFail.Await(ctx)
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if !caught {
		t.Fatal("Catch callback not called")
	}

	t.Log("Error handling test passed")
}

func TestPost(t *testing.T) {
	r, err := antsx.NewReactor(2)
	if err != nil {
		t.Fatalf("Failed to create reactor: %v", err)
	}
	defer r.Release()

	ctx := context.Background()
	called := make(chan struct{}, 1)

	err = antsx.Post(ctx, r, func(ctx context.Context) (string, error) {
		time.Sleep(100 * time.Millisecond)
		called <- struct{}{}
		return "done", nil
	})

	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	select {
	case <-called:
		t.Logf("Post task executed successfully")
	case <-time.After(1 * time.Second):
		t.Fatalf("Post task did not complete in time")
	}
}
