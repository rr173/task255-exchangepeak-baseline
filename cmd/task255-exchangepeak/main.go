package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"task255-exchangepeak/internal/httpapi"
	"task255-exchangepeak/internal/peak"
	"task255-exchangepeak/internal/sample"
	"task255-exchangepeak/internal/service"
	"task255-exchangepeak/internal/store"
)

func main() {
	var (
		dbPath    = flag.String("db", "exchangepeak.db", "SQLite 数据库文件路径")
		addr      = flag.String("addr", ":8080", "HTTP 监听地址")
		smokeTest = flag.Bool("smoke-test", false, "运行端到端自检后退出（验证持久化与重启恢复）")
	)
	flag.Parse()

	if *smokeTest {
		if err := runSmokeTest(*dbPath); err != nil {
			log.Fatalf("SMOKE-TEST-FAILED: %v", err)
		}
		fmt.Println("SMOKE-TEST-OK")
		return
	}

	st, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	svc := service.New(st)
	fmt.Printf("task255-exchangepeak listening on %s (db=%s)\n", *addr, *dbPath)
	if err := http.ListenAndServe(*addr, httpapi.Handler(svc)); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// runSmokeTest 执行端到端流程：创建实体 → 分析 → 关闭 → 重开 → 校验持久化。
func runSmokeTest(dbPath string) error {
	if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clean db: %w", err)
	}
	defer os.Remove(dbPath)

	// 第一轮：写入并分析
	st, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open round1: %w", err)
	}
	svc := service.New(st)

	smp, err := svc.Sample.Create(sample.CreateInput{
		Name:          "ethanol",
		Compound:      "C2H5OH",
		Solvent:       "CDCl3",
		Concentration: 0.5,
	})
	if err != nil {
		return fmt.Errorf("create sample: %w", err)
	}
	temps, err := svc.Sample.AddTemperatures(smp.ID, []float64{25, 50, 75})
	if err != nil {
		return fmt.Errorf("add temps: %w", err)
	}
	tempID := map[float64]string{}
	for _, t := range temps {
		tempID[t.TempC] = t.ID
	}

	batch, err := svc.CreateBatch(smp.ID, "smoke-batch")
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}
	std, err := svc.CreateStandard(batch.ID, "TMS")
	if err != nil {
		return fmt.Errorf("create standard: %w", err)
	}
	for _, t := range temps {
		if err := svc.AddStandardPoint(std.ID, t.TempC, 0.00); err != nil {
			return fmt.Errorf("add std point: %w", err)
		}
	}
	if err := svc.LockStandard(std.ID); err != nil {
		return fmt.Errorf("lock standard: %w", err)
	}

	// 两条共振，随升温向彼此靠拢（化学交换融合） + 每档参比峰
	for _, t := range temps {
		// 参比峰（is_standard=true）
		if _, err := svc.Peak.Add(peak.AddInput{
			BatchID: batch.ID, TemperatureID: tempID[t.TempC], TempC: t.TempC,
			Unit: "ppm", ObservedShift: 0.05, Intensity: 1.0, IsStandard: true,
		}); err != nil {
			return fmt.Errorf("std peak: %w", err)
		}
	}
	observedA := map[float64]float64{25: 1.50, 50: 1.40, 75: 1.30}
	observedB := map[float64]float64{25: 1.80, 50: 1.50, 75: 1.32}
	for _, t := range temps {
		if _, err := svc.Peak.Add(peak.AddInput{
			BatchID: batch.ID, TemperatureID: tempID[t.TempC], TempC: t.TempC,
			Unit: "ppm", ObservedShift: observedA[t.TempC], Intensity: 0.8,
		}); err != nil {
			return fmt.Errorf("peak A: %w", err)
		}
		if _, err := svc.Peak.Add(peak.AddInput{
			BatchID: batch.ID, TemperatureID: tempID[t.TempC], TempC: t.TempC,
			Unit: "ppm", ObservedShift: observedB[t.TempC], Intensity: 0.8,
		}); err != nil {
			return fmt.Errorf("peak B: %w", err)
		}
	}

	if _, err := svc.Peak.Calibrate(batch.ID); err != nil {
		return fmt.Errorf("calibrate: %w", err)
	}
	if _, err := svc.Track.Associate(batch.ID); err != nil {
		return fmt.Errorf("associate: %w", err)
	}
	cands, err := svc.Exchange.Score(batch.ID)
	if err != nil {
		return fmt.Errorf("score: %w", err)
	}
	if len(cands) == 0 {
		return fmt.Errorf("expected at least one exchange candidate, got 0")
	}
	if _, err := svc.Snapshot.Freeze(batch.ID); err != nil {
		return fmt.Errorf("freeze: %w", err)
	}

	if err := st.Close(); err != nil {
		return fmt.Errorf("close round1: %w", err)
	}

	// 第二轮：重开数据库，验证持久化与重启恢复
	st2, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open round2: %w", err)
	}
	defer st2.Close()
	svc2 := service.New(st2)

	stats, err := svc2.SelfCheck()
	if err != nil {
		return fmt.Errorf("selfcheck: %w", err)
	}
	if stats.Samples != 1 || stats.Batches != 1 || stats.Peaks != 9 ||
		stats.Tracks < 2 || stats.Candidates < 1 || stats.Snapshots != 1 {
		return fmt.Errorf("persistence mismatch: %+v", stats)
	}
	if _, err := svc2.Store.GetBatch(batch.ID); err != nil {
		return fmt.Errorf("batch not persisted: %w", err)
	}
	return nil
}
