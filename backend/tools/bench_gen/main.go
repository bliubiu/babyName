// 命令 bench_gen 是起名引擎的性能基准工具。
//
// 用真实数据（data 目录 / namer.json 8105 字）跑一次完整的双名全量枚举生成，
// 输出候选池规模、各阶段耗时，并可选写出 CPU profile 供 pprof 分析。
//
// 用法（在 backend/tools/bench_gen 目录下）：
//
//	go run . -timeout 60s -cpuprofile cpu.prof
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"name/internal/application/services"
	"name/internal/domain/fate"
	"name/internal/infrastructure/cache"
	"name/internal/infrastructure/data"
)

func main() {
	dataDir := flag.String("data", "../../data", "数据目录")
	timeout := flag.Duration("timeout", 60*time.Second, "生成超时（超时即打印 n/a）")
	cpuProfile := flag.String("cpuprofile", "", "CPU profile 输出路径（空则不采样）")
	surname := flag.String("surname", "张", "姓氏")
	gender := flag.String("gender", "male", "性别 male/female")
	count := flag.Int("count", 50, "期望名字数")
	flag.Parse()

	t0 := time.Now()
	if err := data.Init(*dataDir); err != nil {
		fmt.Println("data.Init 失败:", err)
		return
	}
	services.SyncNamingIndexFromHanzi()
	cache.Init()
	fmt.Printf("数据装载耗时: %v\n", time.Since(t0).Round(time.Millisecond))

	provider := &services.HanziDataProvider{}
	analyzer := services.NewBaziAnalyzerAdapter()

	born := time.Date(2024, 5, 20, 10, 0, 0, 0, time.UTC)

	fo := fate.NewFilterOption().WithGenderFilter(*gender).WithStrictness("moderate")
	filter := fo.Build()
	var q fate.CharacterQuery = fate.NewBasicCharacterQuery()
	q = filter.QueryFilter(q)
	all, _ := provider.FindCharacters(q)
	fmt.Printf("候选池: filter.QueryFilter 后 %d 字\n", len(all))

	fd, err := analyzer.Analyze(born, fate.Gender(*gender))
	if err != nil {
		fmt.Println("八字分析失败:", err)
		return
	}
	fmt.Printf("喜用神: %v  日主: %s %s\n",
		fd.WuXingXiji.XiYongShen, fd.WuXingXiji.RiZhu, fd.WuXingXiji.RiZhuWuXing)

	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			fmt.Println("创建 profile 失败:", err)
			return
		}
		_ = pprof.StartCPUProfile(f)
		defer func() { pprof.StopCPUProfile(); _ = f.Close() }()
	}

	fmt.Printf("CPU 核数: %d\n", runtime.NumCPU())
	fmt.Printf("开始生成: %s%s %s Count=%d ...\n", *surname, "○○", *gender, *count)

	engine := fate.NewEngine(provider, analyzer, fate.DefaultRaters())
	sess := engine.NewSessionWithFilter(filter)
	input := fate.Input{
		Surname: *surname,
		Born:    born,
		Gender:  fate.Gender(*gender),
		Options: fate.GenerateOptions{Count: *count, NameLength: 2},
	}

	start := time.Now()
	if err := sess.Start(context.Background(), &input); err != nil {
		fmt.Println("启动失败:", err)
		return
	}

	done := make(chan error, 1)
	go func() { done <- sess.Wait() }()

	select {
	case err := <-done:
		elapsed := time.Since(start)
		if err != nil {
			fmt.Printf("生成失败: %v（耗时 %v）\n", err, elapsed.Round(time.Millisecond))
			return
		}
		out := sess.Result()
		fmt.Printf("生成完成: 耗时 %v，已评分组合数 %d，返回名字 %d 个\n",
			elapsed.Round(time.Millisecond), out.TotalCount, len(out.TopNames))
		for i, n := range out.TopNames {
			if i >= 10 {
				break
			}
			fmt.Printf("  %2d. %s%s %.1f\n", i+1, *surname, n.GivenName, n.Score.Total)
		}
	case <-time.After(*timeout):
		fmt.Printf("超时！%v 内未完成（已取消会话）\n", *timeout)
		_ = sess.Stop()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}
}
