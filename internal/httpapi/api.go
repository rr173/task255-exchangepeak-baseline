package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"task255-exchangepeak/internal/model"
	"task255-exchangepeak/internal/peak"
	"task255-exchangepeak/internal/sample"
	"task255-exchangepeak/internal/service"
)

// Handler 返回挂载全部 /api 路由的 http.Handler。
func Handler(svc *service.Service) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/selfcheck", func(w http.ResponseWriter, r *http.Request) {
		st, err := svc.SelfCheck()
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, st)
	})

	// 样品
	mux.HandleFunc("POST /api/samples", func(w http.ResponseWriter, r *http.Request) {
		var in sample.CreateInput
		if err := readJSON(r, &in); err != nil {
			writeErr(w, model.ErrInvalidInput)
			return
		}
		smp, err := svc.Sample.Create(in)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, smp)
	})
	mux.HandleFunc("GET /api/samples", func(w http.ResponseWriter, r *http.Request) {
		list, err := svc.Sample.List()
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	})
	mux.HandleFunc("GET /api/samples/{id}", func(w http.ResponseWriter, r *http.Request) {
		smp, err := svc.Sample.Get(r.PathValue("id"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, smp)
	})
	mux.HandleFunc("GET /api/samples/{id}/temperatures", func(w http.ResponseWriter, r *http.Request) {
		list, err := svc.Sample.Temperatures(r.PathValue("id"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	})
	mux.HandleFunc("POST /api/samples/{id}/temperatures", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Temps []float64 `json:"temps"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, model.ErrInvalidInput)
			return
		}
		out, err := svc.Sample.AddTemperatures(r.PathValue("id"), body.Temps)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	})
	mux.HandleFunc("DELETE /api/samples/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := svc.Sample.Delete(r.PathValue("id")); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
	})

	// 谱图批次
	mux.HandleFunc("POST /api/batches", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			SampleID string  `json:"sample_id"`
			Label    string  `json:"label"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, model.ErrInvalidInput)
			return
		}
		b, err := svc.CreateBatch(body.SampleID, body.Label)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, b)
	})
	mux.HandleFunc("GET /api/batches", func(w http.ResponseWriter, r *http.Request) {
		sampleID := r.URL.Query().Get("sample_id")
		list, err := svc.Store.ListBatches(sampleID)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	})
	mux.HandleFunc("GET /api/batches/{id}", func(w http.ResponseWriter, r *http.Request) {
		b, err := svc.Store.GetBatch(r.PathValue("id"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, b)
	})
	mux.HandleFunc("POST /api/batches/{id}/state", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			State string `json:"state"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, model.ErrInvalidInput)
			return
		}
		if err := svc.SetBatchState(r.PathValue("id"), body.State); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"state": body.State})
	})

	// 峰
	mux.HandleFunc("POST /api/batches/{id}/peaks", func(w http.ResponseWriter, r *http.Request) {
		var in peak.AddInput
		if err := readJSON(r, &in); err != nil {
			writeErr(w, model.ErrInvalidInput)
			return
		}
		in.BatchID = r.PathValue("id")
		p, err := svc.Peak.Add(in)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, p)
	})
	mux.HandleFunc("GET /api/batches/{id}/peaks", func(w http.ResponseWriter, r *http.Request) {
		list, err := svc.Peak.List(r.PathValue("id"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	})
	mux.HandleFunc("GET /api/batches/{id}/peaks/temp", func(w http.ResponseWriter, r *http.Request) {
		t, err := strconv.ParseFloat(r.URL.Query().Get("temp"), 64)
		if err != nil {
			writeErr(w, model.ErrInvalidInput)
			return
		}
		list, err := svc.Peak.ListAtTemp(r.PathValue("id"), t)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	})
	mux.HandleFunc("POST /api/batches/{id}/peaks/{peakId}/impurity", func(w http.ResponseWriter, r *http.Request) {
		if err := svc.Peak.MarkImpurity(r.PathValue("peakId")); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"marked": true})
	})
	mux.HandleFunc("POST /api/batches/{id}/peaks/{peakId}/exclude", func(w http.ResponseWriter, r *http.Request) {
		if err := svc.Peak.MarkExcluded(r.PathValue("peakId")); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"excluded": true})
	})

	// 内标
	mux.HandleFunc("POST /api/batches/{id}/standards", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name string `json:"name"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, model.ErrInvalidInput)
			return
		}
		std, err := svc.CreateStandard(r.PathValue("id"), body.Name)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, std)
	})
	mux.HandleFunc("POST /api/standards/{id}/points", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			TempC     float64 `json:"temp_c"`
			TrueShift float64 `json:"true_shift"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, model.ErrInvalidInput)
			return
		}
		if err := svc.AddStandardPoint(r.PathValue("id"), body.TempC, body.TrueShift); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]bool{"added": true})
	})
	mux.HandleFunc("POST /api/standards/{id}/lock", func(w http.ResponseWriter, r *http.Request) {
		if err := svc.LockStandard(r.PathValue("id")); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"locked": true})
	})

	// 分析流水线
	mux.HandleFunc("POST /api/batches/{id}/calibrate", func(w http.ResponseWriter, r *http.Request) {
		offsets, err := svc.Peak.Calibrate(r.PathValue("id"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, offsets)
	})
	mux.HandleFunc("POST /api/batches/{id}/associate", func(w http.ResponseWriter, r *http.Request) {
		list, err := svc.Track.Associate(r.PathValue("id"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	})
	mux.HandleFunc("POST /api/batches/{id}/score", func(w http.ResponseWriter, r *http.Request) {
		list, err := svc.Exchange.Score(r.PathValue("id"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	})
	mux.HandleFunc("POST /api/batches/{id}/analyze", func(w http.ResponseWriter, r *http.Request) {
		if err := svc.Analyze(r.PathValue("id")); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"analyzed": true})
	})

	// 轨迹
	mux.HandleFunc("GET /api/batches/{id}/tracks", func(w http.ResponseWriter, r *http.Request) {
		list, err := svc.Track.List(r.PathValue("id"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	})
	mux.HandleFunc("GET /api/tracks/{id}", func(w http.ResponseWriter, r *http.Request) {
		t, err := svc.Track.Get(r.PathValue("id"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, t)
	})
	mux.HandleFunc("GET /api/tracks/{id}/members", func(w http.ResponseWriter, r *http.Request) {
		list, err := svc.Track.Members(r.PathValue("id"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	})

	// 交换候选
	mux.HandleFunc("GET /api/batches/{id}/candidates", func(w http.ResponseWriter, r *http.Request) {
		list, err := svc.Exchange.List(r.PathValue("id"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	})
	mux.HandleFunc("GET /api/candidates/{id}", func(w http.ResponseWriter, r *http.Request) {
		c, err := svc.Exchange.Get(r.PathValue("id"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, c)
	})
	mux.HandleFunc("POST /api/candidates/{id}/confirm", func(w http.ResponseWriter, r *http.Request) {
		if err := svc.Exchange.Confirm(r.PathValue("id")); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"confirmed": true})
	})
	mux.HandleFunc("POST /api/candidates/{id}/reject", func(w http.ResponseWriter, r *http.Request) {
		if err := svc.Exchange.Reject(r.PathValue("id")); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"rejected": true})
	})

	// 归属快照
	mux.HandleFunc("POST /api/batches/{id}/snapshots", func(w http.ResponseWriter, r *http.Request) {
		snap, err := svc.Snapshot.Freeze(r.PathValue("id"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, snap)
	})
	mux.HandleFunc("GET /api/batches/{id}/snapshots", func(w http.ResponseWriter, r *http.Request) {
		list, err := svc.Snapshot.List(r.PathValue("id"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	})
	mux.HandleFunc("GET /api/snapshots/{id}", func(w http.ResponseWriter, r *http.Request) {
		snap, err := svc.Snapshot.Get(r.PathValue("id"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, snap)
	})

	return mux
}

func readJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	return dec.Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": v})
}

func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, model.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, model.ErrInvalidInput),
		errors.Is(err, model.ErrTempConflict),
		errors.Is(err, model.ErrInvalidState),
		errors.Is(err, model.ErrMissingStandard),
		errors.Is(err, model.ErrNoStandardPeak),
		errors.Is(err, model.ErrUnitMismatch):
		status = http.StatusBadRequest
	case errors.Is(err, model.ErrSealedBatch),
		errors.Is(err, model.ErrStandardLocked):
		status = http.StatusConflict
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
}
