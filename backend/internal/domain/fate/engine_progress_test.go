package fate

import (
	"context"
	"testing"
	"time"
)

// TestSessionProgress_ProgressReporter 会话必须实现 ProgressReporter，
// 且生成完成后进度应到 100%、阶段为「汇总排序」。
func TestSessionProgress_ProgressReporter(t *testing.T) {
	engine := newPinyinTestEngine()
	session := engine.NewSession()
	pr, ok := session.(ProgressReporter)
	if !ok {
		t.Fatal("sessionImpl 应实现 ProgressReporter 接口")
	}

	if err := session.Start(context.Background(), &Input{
		Surname: "李",
		Gender:  GenderMale,
		Born:    time.Date(2023, 8, 20, 10, 0, 0, 0, time.UTC),
		Options: GenerateOptions{NameLength: 2, Count: 10},
	}); err != nil {
		t.Fatalf("会话启动失败: %v", err)
	}

	// 生成中轮询：百分比应在 [0,100] 内（可能很快结束，仅做越界防御）
	if stage, pct := pr.Progress(); pct < 0 || pct > 100 {
		t.Fatalf("进度越界: stage=%s pct=%v", stage, pct)
	}

	if err := session.Wait(); err != nil {
		t.Fatalf("生成失败: %v", err)
	}

	stage, pct := pr.Progress()
	if pct != 100 {
		t.Fatalf("完成后进度应为 100，got %v", pct)
	}
	if stage != "汇总排序" {
		t.Fatalf("完成后的阶段应为 汇总排序，got %s", stage)
	}
}

// TestSessionProgress_Monotonic 双名枚举期间进度单调不减（采集线程持续轮询）
func TestSessionProgress_Monotonic(t *testing.T) {
	engine := newPinyinTestEngine()
	session := engine.NewSession()
	pr, ok := session.(ProgressReporter)
	if !ok {
		t.Fatal("sessionImpl 应实现 ProgressReporter 接口")
	}
	if err := session.Start(context.Background(), &Input{
		Surname: "李",
		Gender:  GenderMale,
		Born:    time.Date(2023, 8, 20, 10, 0, 0, 0, time.UTC),
		Options: GenerateOptions{NameLength: 2, Count: 10},
	}); err != nil {
		t.Fatalf("会话启动失败: %v", err)
	}

	prev := -1.0
	for session.State() == SessionStateGenerating {
		_, pct := pr.Progress()
		if pct < prev {
			t.Fatalf("进度回退: %v < %v", pct, prev)
		}
		prev = pct
		time.Sleep(50 * time.Microsecond)
	}
	if err := session.Wait(); err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	if _, pct := pr.Progress(); pct < prev {
		t.Fatalf("完成进度应不低于枚举期峰值: %v < %v", pct, prev)
	}
}
