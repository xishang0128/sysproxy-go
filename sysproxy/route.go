//go:build darwin

package sysproxy

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type Request struct {
	Server string `json:"server"`
	Bypass string `json:"bypass"`
	Url    string `json:"url"`
}

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func router() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/*", status)
	r.Post("/pac", pac)
	r.Post("/proxy", proxy)
	r.Post("/disable", disable)
	return r
}

func status(w http.ResponseWriter, r *http.Request) {
	status, err := QueryProxySettings()
	if err != nil {
		sendError(w, err)
		return
	}
	render.JSON(w, r, status)
}

func pac(w http.ResponseWriter, r *http.Request) {
	var req Request
	if err := decodeRequest(r, &req); err != nil {
		sendError(w, err)
		return
	}

	err := SetPac(req.Url)
	if err != nil {
		sendError(w, err)
		return
	}
	render.NoContent(w, r)
}

func proxy(w http.ResponseWriter, r *http.Request) {
	var req Request
	if err := decodeRequest(r, &req); err != nil {
		sendError(w, err)
		return
	}

	err := SetProxy(req.Server, req.Bypass)
	if err != nil {
		sendError(w, err)
		return
	}
	render.NoContent(w, r)
}

func disable(w http.ResponseWriter, r *http.Request) {
	err := DisableProxy()
	if err != nil {
		sendError(w, err)
		return
	}
	render.NoContent(w, r)
}

func decodeRequest(r *http.Request, v any) error {
	if r.ContentLength > 0 {
		return render.DecodeJSON(r.Body, v)
	}
	return nil
}

func sendJSON(w http.ResponseWriter, status string, message string) {
	w.Header().Set("Content-Type", "application/json")
	resp := Response{
		Status:  status,
		Message: message,
	}
	json.NewEncoder(w).Encode(resp)
}

func sendError(w http.ResponseWriter, err error) {
	sendJSON(w, "error", err.Error())
}
