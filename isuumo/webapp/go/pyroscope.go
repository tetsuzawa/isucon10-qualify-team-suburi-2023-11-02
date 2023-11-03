package main

import (
	"runtime"

	"github.com/grafana/pyroscope-go"
)

func startPyroscope() *pyroscope.Profiler {
	mutexProfileRate := 1
	runtime.SetMutexProfileFraction(mutexProfileRate) // ・・・(1)
	blockProfileRate := 1
	runtime.SetBlockProfileRate(blockProfileRate) // ・・・(2)
	p, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: "isuumo",
		ServerAddress:   GetEnv("PYROSCOPE_SERVER", "http://monitoring:4040"),
		Logger:          pyroscope.StandardLogger,

		// タグを設定することで、タグ指定でのプロファイル表示や、タグ間のプロファイル比較ができ便利です
		Tags: map[string]string{
			"hostname": "isuumo",
			"version":  GetEnv("APP_VERSION", "000000"),
		},

		ProfileTypes: []pyroscope.ProfileType{
			// デフォルトで取得するプロファイル
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,

			// オプショナルで取得するプロファイル
			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	})
	if err != nil {
		panic(err)
	}
	return p
}
